package seo

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"strings"

	"github.com/goplaid/web"
	"github.com/goplaid/x/i18n"
	"github.com/goplaid/x/perm"
	"github.com/goplaid/x/presets"
	. "github.com/goplaid/x/vuetify"
	"github.com/qor/qor5/media"
	"github.com/qor/qor5/media/media_library"
	"github.com/qor/qor5/media/views"
	"github.com/sunfmin/reflectutils"
	h "github.com/theplant/htmlgo"
	"golang.org/x/text/language"
	"gorm.io/gorm"
)

const (
	saveEvent                 = "seo_save_collection"
	I18nSeoKey i18n.ModuleKey = "I18nSeoKey"
)

var permVerifier *perm.Verifier

func (collection *Collection) Configure(b *presets.Builder, db *gorm.DB) {
	if err := db.AutoMigrate(collection.settingModel); err != nil {
		panic(err)
	}

	if GlobalDB == nil {
		GlobalDB = db
	}

	b.GetWebBuilder().RegisterEventFunc(saveEvent, collection.save)
	b.Model(collection.settingModel).PrimaryField("Name").Label("SEO").Listing().PageFunc(collection.pageFunc)

	b.FieldDefaults(presets.WRITE).
		FieldType(Setting{}).
		ComponentFunc(collection.editingComponentFunc)

	b.I18n().
		RegisterForModule(language.English, I18nSeoKey, Messages_en_US).
		RegisterForModule(language.SimplifiedChinese, I18nSeoKey, Messages_zh_CN)

	b.ExtraAsset("/vue-seo.js", "text/javascript", SeoJSComponentsPack())
	permVerifier = perm.NewVerifier("seo", b.GetPermission())
}

func (collection *Collection) editingComponentFunc(obj interface{}, field *presets.FieldContext, ctx *web.EventContext) h.HTMLComponent {
	var (
		msgr        = i18n.MustGetModuleMessages(ctx.R, I18nSeoKey, Messages_en_US).(*Messages)
		fieldPrefix string
		setting     Setting
	)

	seo := collection.GetSEOByModel(obj)
	if seo == nil {
		return h.Div()
	}

	value := reflect.Indirect(reflect.ValueOf(obj))
	for i := 0; i < value.NumField(); i++ {
		if s, ok := value.Field(i).Interface().(Setting); ok {
			setting = s
			fieldPrefix = value.Type().Field(i).Name
		}
	}

	return h.HTMLComponents{
		h.Label(msgr.Seo).Class("v-label theme--light"),
		web.Scope(
			VCard(
				VCardText(
					VSwitch().Label(msgr.UseDefaults).Attr("v-model", "locals.userDefaults").On("change", "locals.enabledCustomize = !locals.userDefaults;$refs.customize.$emit('change', locals.enabledCustomize)"),
					VSwitch().FieldName(fmt.Sprintf("%s.%s", fieldPrefix, "EnabledCustomize")).Value(setting.EnabledCustomize).Attr(":input-value", "locals.enabledCustomize").Attr("ref", "customize").Attr("style", "display:none;"),
					h.Div(collection.vseo(fieldPrefix, seo, &setting, ctx.R)).Attr("v-show", "locals.userDefaults == false"),
				),
			).Attr("style", "margin-bottom: 15px; margin-top: 15px;"),
		).Init(fmt.Sprintf(`{enabledCustomize: %t, userDefaults: %t}`, setting.EnabledCustomize, !setting.EnabledCustomize)).
			VSlot("{ locals }"),
	}
}

func (collection *Collection) pageFunc(ctx *web.EventContext) (_ web.PageResponse, err error) {
	var (
		msgr = i18n.MustGetModuleMessages(ctx.R, I18nSeoKey, Messages_en_US).(*Messages)
		db   = collection.getDBFromContext(ctx.R.Context())
	)

	var seoComponents h.HTMLComponents
	for _, seo := range collection.registeredSEO {
		modelSetting := collection.NewSettingModelInstance().(QorSEOSettingInterface)
		err := db.Where("name = ?", seo.name).First(modelSetting).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			modelSetting.SetName(seo.name)
			if err := db.Save(modelSetting).Error; err != nil {
				panic(err)
			}
		}

		var variablesComps h.HTMLComponents
		if seo.settingVariables != nil {
			variables := reflect.Indirect(reflect.New(reflect.Indirect(reflect.ValueOf(seo.settingVariables)).Type()))
			variableValues := modelSetting.GetVariables()
			for i := 0; i < variables.NumField(); i++ {
				fieldName := variables.Type().Field(i).Name
				if variableValues[fieldName] != "" {
					fmt.Println(fieldName, variableValues[fieldName])
					variables.Field(i).Set(reflect.ValueOf(variableValues[fieldName]))
				}
			}

			if variables.Type().NumField() > 0 {
				variablesComps = append(variablesComps, h.H3(msgr.Variable).Style("margin-top:15px;font-weight: 500"))
			}

			for i := 0; i < variables.Type().NumField(); i++ {
				field := variables.Type().Field(i)
				variablesComps = append(variablesComps, VTextField().FieldName(fmt.Sprintf("%s.Variables.%s", seo.name, field.Name)).Label(i18n.PT(ctx.R, presets.ModelsI18nModuleKey, "Seo Variable", field.Name)).Value(variables.Field(i).String()))
			}
		}

		var (
			label       string
			setting     = modelSetting.GetSEOSetting()
			loadingName = strings.ReplaceAll(strings.ToLower(seo.name), " ", "_")
		)

		if seo.name == collection.globalName {
			label = msgr.GlobalName
		} else {
			label = i18n.PT(ctx.R, presets.ModelsI18nModuleKey, "Seo", seo.name)
		}

		comp := VExpansionPanel(
			VExpansionPanelHeader(h.H4(label).Style("font-weight: 500;")),
			VExpansionPanelContent(
				VCardText(
					variablesComps,
					collection.vseo(modelSetting.GetName(), seo, &setting, ctx.R),
				),
				VCardActions(
					VSpacer(),
					VBtn(msgr.Save).Bind("loading", fmt.Sprintf("vars.%s", loadingName)).Color("primary").Large(true).
						Attr("@click", web.Plaid().
							EventFunc(saveEvent).
							Query("name", seo.name).
							Query("loadingName", loadingName).
							BeforeScript(fmt.Sprintf("vars.%s = true;", loadingName)).Go()),
				).Attr(web.InitContextVars, fmt.Sprintf(`{%s: false}`, loadingName)),
			),
		)
		seoComponents = append(seoComponents, comp)
	}

	return web.PageResponse{
		PageTitle: msgr.PageTitle,
		Body: h.If(editIsAllowed(ctx.R) == nil, VContainer(
			VSnackbar(h.Text(msgr.SavedSuccessfully)).
				Attr("v-model", "vars.seoSnackbarShow").
				Top(true).
				Color("primary").
				Timeout(2000),
			VRow(
				VCol(
					VContainer(
						h.H3(msgr.PageMetadataTitle).Attr("style", "font-weight: 500"),
						h.P().Text(msgr.PageMetadataDescription)),
				).Cols(3),
				VCol(
					VExpansionPanels(
						seoComponents,
					).Focusable(true),
				).Cols(9),
			),
		).Attr("style", "background-color: #f5f5f5;max-width:100%").Attr(web.InitContextVars, `{seoSnackbarShow: false}`)),
	}, nil
}

func (collection *Collection) vseo(fieldPrefix string, seo *SEO, setting *Setting, req *http.Request) h.HTMLComponent {
	var (
		seos []*SEO
		msgr = i18n.MustGetModuleMessages(req, I18nSeoKey, Messages_en_US).(*Messages)
		db   = collection.getDBFromContext(req.Context())
	)

	if seo.name == collection.globalName {
		seos = append(seos, seo)
	} else {
		seos = append(seos, collection.GetSEO(collection.globalName), seo)
	}

	var (
		variablesEle []h.HTMLComponent
		variables    []string
	)

	for _, seo := range seos {
		if seo.settingVariables != nil {
			value := reflect.Indirect(reflect.ValueOf(seo.settingVariables)).Type()
			for i := 0; i < value.NumField(); i++ {
				fieldName := value.Field(i).Name
				variables = append(variables, fieldName)
			}
		}

		for key := range seo.contextVariables {
			if !strings.Contains(key, ":") {
				variables = append(variables, key)
			}
		}
	}

	for _, variable := range variables {
		variablesEle = append(variablesEle, VCol(
			VBtn("").Width(100).Icon(true).Attr("style", "text-transform: none").Children(VIcon("add_box"), h.Text(i18n.PT(req, presets.ModelsI18nModuleKey, "Seo Variable", variable))).Attr("@click", fmt.Sprintf("$refs.seo.addTags('%s')", variable)),
		).Cols(2))
	}

	image := &setting.OpenGraphImageFromMediaLibrary
	if image.ID.String() == "0" {
		image.ID = json.Number("")
	}
	refPrefix := strings.ReplaceAll(strings.ToLower(fieldPrefix), " ", "_")
	return VSeo(
		h.H4(msgr.Basic).Style("margin-top:15px;font-weight: 500"),
		VRow(
			variablesEle...,
		),
		VCard(
			VCardText(
				VTextField().Counter(65).FieldName(fmt.Sprintf("%s.%s", fieldPrefix, "Title")).Label(msgr.Title).Value(setting.Title).Attr("@click", fmt.Sprintf("$refs.seo.tagInputsFocus($refs.%s)", fmt.Sprintf("%s_title", refPrefix))).Attr("ref", fmt.Sprintf("%s_title", refPrefix)),
				VTextField().Counter(150).FieldName(fmt.Sprintf("%s.%s", fieldPrefix, "Description")).Label(msgr.Description).Value(setting.Description).Attr("@click", fmt.Sprintf("$refs.seo.tagInputsFocus($refs.%s)", fmt.Sprintf("%s_description", refPrefix))).Attr("ref", fmt.Sprintf("%s_description", refPrefix)),
				VTextarea().Counter(255).Rows(2).AutoGrow(true).FieldName(fmt.Sprintf("%s.%s", fieldPrefix, "Keywords")).Label(msgr.Keywords).Value(setting.Keywords).Attr("@click", fmt.Sprintf("$refs.seo.tagInputsFocus($refs.%s)", fmt.Sprintf("%s_keywords", refPrefix))).Attr("ref", fmt.Sprintf("%s_keywords", refPrefix)),
			),
		),

		h.H4(msgr.OpenGraphInformation).Style("margin-top:15px;margin-bottom:15px;font-weight: 500"),
		VCard(
			VCardText(
				VRow(
					VCol(VTextField().FieldName(fmt.Sprintf("%s.%s", fieldPrefix, "OpenGraphURL")).Label(msgr.OpenGraphURL).Value(setting.OpenGraphURL)).Cols(6),
					VCol(VTextField().FieldName(fmt.Sprintf("%s.%s", fieldPrefix, "OpenGraphType")).Label(msgr.OpenGraphType).Value(setting.OpenGraphType)).Cols(6),
				),
				VRow(
					VCol(VTextField().FieldName(fmt.Sprintf("%s.%s", fieldPrefix, "OpenGraphImageURL")).Label(msgr.OpenGraphImageURL).Value(setting.OpenGraphImageURL)).Cols(12),
				),
				VRow(
					VCol(views.QMediaBox(db).Label(msgr.OpenGraphImage).
						FieldName(fmt.Sprintf("%s.%s", fieldPrefix, "OpenGraphImageFromMediaLibrary")).
						Value(image).
						Config(&media_library.MediaBoxConfig{
							AllowType: "image",
							Sizes: map[string]*media.Size{
								"og": {
									Width:  1200,
									Height: 630,
								},
								"twitter-large": {
									Width:  1200,
									Height: 600,
								},
								"twitter-small": {
									Width:  630,
									Height: 630,
								},
							},
						})).Cols(12)),
			),
		),
	).Attr("ref", "seo")
}

func (collection *Collection) save(ctx *web.EventContext) (r web.EventResponse, err error) {
	var (
		db      = collection.getDBFromContext(ctx.R.Context())
		name    = ctx.R.FormValue("name")
		setting = collection.NewSettingModelInstance().(QorSEOSettingInterface)
	)

	if err = db.Where("name = ?", name).First(setting).Error; err != nil {
		return
	}

	var (
		variables   = map[string]string{}
		settingVals = map[string]interface{}{}
		mediaBox    = media_library.MediaBox{}
	)

	for fieldWithPrefix := range ctx.R.Form {
		if !strings.HasPrefix(fieldWithPrefix, name) {
			continue
		}

		field := strings.Replace(fieldWithPrefix, fmt.Sprintf("%s.", name), "", -1)
		if strings.HasPrefix(field, "OpenGraphImageFromMediaLibrary") {
			if field == "OpenGraphImageFromMediaLibrary.Values" {
				err = mediaBox.Scan(ctx.R.FormValue(fieldWithPrefix))
				if err != nil {
					return
				}
				settingVals["OpenGraphImageFromMediaLibrary"] = mediaBox
			}

			if field == "OpenGraphImageFromMediaLibrary.Description" {
				mediaBox.Description = ctx.R.FormValue(fieldWithPrefix)
				if err != nil {
					return
				}
			}
		} else if strings.HasPrefix(field, "Variables") {
			key := strings.Replace(field, "Variables.", "", -1)
			variables[key] = ctx.R.FormValue(fieldWithPrefix)
		} else {
			settingVals[field] = ctx.R.Form.Get(fieldWithPrefix)
		}
	}
	s := setting.GetSEOSetting()
	for k, v := range settingVals {
		err = reflectutils.Set(&s, k, v)
		if err != nil {
			return
		}
	}

	setting.SetSEOSetting(s)
	setting.SetVariables(variables)
	if err = db.Save(setting).Error; err != nil {
		return
	}
	r.VarsScript = fmt.Sprintf(`vars.seoSnackbarShow = true;vars.%s = false;`, ctx.R.FormValue("loadingName"))
	return
}
