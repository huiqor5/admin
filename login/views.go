package login

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image/png"
	"net/http"
	"net/url"

	"github.com/goplaid/web"
	"github.com/goplaid/x/i18n"
	"github.com/pquerna/otp"
	. "github.com/theplant/htmlgo"
	"golang.org/x/text/language"
	"golang.org/x/text/language/display"
	"gorm.io/gorm"
)

func getFailCodeText(msgr *Messages, code FailCode) string {
	switch code {
	case FailCodeSystemError:
		return msgr.ErrorSystemError
	case FailCodeCompleteUserAuthFailed:
		return msgr.ErrorCompleteUserAuthFailed
	case FailCodeUserNotFound:
		return msgr.ErrorUserNotFound
	case FailCodeIncorrectAccountNameOrPassword:
		return msgr.ErrorIncorrectAccountNameOrPassword
	case FailCodeUserLocked:
		return msgr.ErrorUserLocked
	case FailCodeAccountIsRequired:
		return msgr.ErrorAccountIsRequired
	case FailCodePasswordCannotBeEmpty:
		return msgr.ErrorPasswordCannotBeEmpty
	case FailCodePasswordNotMatch:
		return msgr.ErrorPasswordNotMatch
	case FailCodeIncorrectPassword:
		return msgr.ErrorIncorrectPassword
	case FailCodeInvalidToken:
		return msgr.ErrorInvalidToken
	case FailCodeTokenExpired:
		return msgr.ErrorTokenExpired
	case FailCodeIncorrectTOTP:
		return msgr.ErrorIncorrectTOTP
	case FailCodeIncorrectRecaptchaToken:
		return msgr.ErrorIncorrectRecaptchaToken
	}

	return ""
}

func getWarnCodeText(msgr *Messages, code WarnCode) string {
	switch code {
	case WarnCodePasswordHasBeenChanged:
		return msgr.WarnPasswordHasBeenChanged
	}
	return ""
}

func getInfoCodeText(msgr *Messages, code InfoCode) string {
	switch code {
	case InfoCodePasswordSuccessfullyReset:
		return msgr.InfoPasswordSuccessfullyReset
	case InfoCodePasswordSuccessfullyChanged:
		return msgr.InfoPasswordSuccessfullyChanged
	}
	return ""
}

const (
	otpKeyFormat = "otpauth://totp/%s:%s?issuer=%s&secret=%s"
)

func errNotice(msg string) HTMLComponent {
	if msg == "" {
		return nil
	}

	return Div().Class("bg-red-100 border border-red-400 text-red-700 px-4 py-3 rounded relative text-center").
		Role("alert").
		Children(
			Span(msg).Class("block sm:inline"),
		)
}

func warnNotice(msg string) HTMLComponent {
	if msg == "" {
		return nil
	}

	return Div().Class("bg-orange-100 border border-orange-400 text-orange-700 px-4 py-3 rounded relative text-center").
		Role("alert").
		Children(
			Span(msg).Class("block sm:inline"),
		)
}

func infoNotice(msg string) HTMLComponent {
	if msg == "" {
		return nil
	}

	return Div().Class("bg-blue-100 border border-blue-400 text-blue-700 px-4 py-3 rounded relative text-center").
		Role("alert").
		Children(
			Span(msg).Class("block sm:inline"),
		)
}

func errorBody(msg string) HTMLComponent {
	return Div(
		Text(msg),
	)
}

func passwordInputWithRevealFunction(
	name string,
	placeholder string,
	id string,
	val string,
) HTMLComponent {
	return Div(
		Input(name).Placeholder(placeholder).Type("password").Class("block w-full px-4 py-2 pr-10 mt-2 text-gray-700 placeholder-gray-400 bg-white border border-gray-200 rounded-md dark:placeholder-gray-600 dark:bg-gray-900 dark:text-gray-300 dark:border-gray-700 focus:border-blue-400 dark:focus:border-blue-400 focus:ring-blue-400 focus:outline-none focus:ring focus:ring-opacity-40").Id(id).
			Value(val),
		Div(
			RawHTML(fmt.Sprintf(`<svg class="h-6 text-gray-700 block" id="icon-%s-showed" fill="none" xmlns="http://www.w3.org/2000/svg" viewbox="0 0 576 512" width="1rem">
  <path fill="currentColor"
    d="M572.52 241.4C518.29 135.59 410.93 64 288 64S57.68 135.64 3.48 241.41a32.35 32.35 0 0 0 0 29.19C57.71 376.41 165.07 448 288 448s230.32-71.64 284.52-177.41a32.35 32.35 0 0 0 0-29.19zM288 400a144 144 0 1 1 144-144 143.93 143.93 0 0 1-144 144zm0-240a95.31 95.31 0 0 0-25.31 3.79 47.85 47.85 0 0 1-66.9 66.9A95.78 95.78 0 1 0 288 160z">
  </path>
</svg>`, id)),
			RawHTML(fmt.Sprintf(`<svg class="h-6 text-gray-700 hidden" id="icon-%s-hidden" fill="none" xmlns="http://www.w3.org/2000/svg" viewbox="0 0 640 512" width="1rem">
  <path fill="currentColor"
    d="M320 400c-75.85 0-137.25-58.71-142.9-133.11L72.2 185.82c-13.79 17.3-26.48 35.59-36.72 55.59a32.35 32.35 0 0 0 0 29.19C89.71 376.41 197.07 448 320 448c26.91 0 52.87-4 77.89-10.46L346 397.39a144.13 144.13 0 0 1-26 2.61zm313.82 58.1l-110.55-85.44a331.25 331.25 0 0 0 81.25-102.07 32.35 32.35 0 0 0 0-29.19C550.29 135.59 442.93 64 320 64a308.15 308.15 0 0 0-147.32 37.7L45.46 3.37A16 16 0 0 0 23 6.18L3.37 31.45A16 16 0 0 0 6.18 53.9l588.36 454.73a16 16 0 0 0 22.46-2.81l19.64-25.27a16 16 0 0 0-2.82-22.45zm-183.72-142l-39.3-30.38A94.75 94.75 0 0 0 416 256a94.76 94.76 0 0 0-121.31-92.21A47.65 47.65 0 0 1 304 192a46.64 46.64 0 0 1-1.54 10l-73.61-56.89A142.31 142.31 0 0 1 320 112a143.92 143.92 0 0 1 144 144c0 21.63-5.29 41.79-13.9 60.11z">
  </path>
</svg>`, id)),
		).Class("absolute right-0 inset-y-0 px-2 flex items-center text-sm cursor-pointer").Id(fmt.Sprintf("btn-reveal-%s", id)),
		Script(fmt.Sprintf(`
(function(){
    var passElem = document.getElementById("%s");
    var revealBtn = document.getElementById("btn-reveal-%s");
    var showedIcon = document.getElementById("icon-%s-showed");
    var hiddenIcon = document.getElementById("icon-%s-hidden");
    revealBtn.onclick = function() {
        if (passElem.type === "password") {
            passElem.type = "text";
            showedIcon.classList.remove("block");
            showedIcon.classList.add("hidden");
            hiddenIcon.classList.remove("hidden");
            hiddenIcon.classList.add("block");
        } else {
            passElem.type = "password";
            hiddenIcon.classList.remove("block");
            hiddenIcon.classList.add("hidden");
            showedIcon.classList.remove("hidden");
            showedIcon.classList.add("block");
        }
    };
})();`, id, id, id, id)),
	).Class("relative")
}

func defaultLoginPage(b *Builder) web.PageFunc {
	return func(ctx *web.EventContext) (r web.PageResponse, err error) {
		// i18n start
		msgr := i18n.MustGetModuleMessages(ctx.R, I18nLoginKey, Messages_en_US).(*Messages)
		var languagesHTML []HTMLComponent
		languages := b.i18nBuilder.GetSupportLanguages()
		if len(languages) > 1 {
			qn := b.i18nBuilder.GetQueryName()
			lang := ctx.R.FormValue(qn)
			if lang == "" {
				lang = b.i18nBuilder.GetCurrentLangFromCookie(ctx.R)
			}
			accept := ctx.R.Header.Get("Accept-Language")
			_, mi := language.MatchStrings(language.NewMatcher(languages), lang, accept)
			for i, l := range languages {
				u, _ := url.Parse(ctx.R.RequestURI)
				qs := u.Query()
				qs.Set(qn, l.String())
				u.RawQuery = qs.Encode()
				elem := Option(display.Self.Name(l)).
					Value(u.String())
				if i == mi {
					elem.Attr("selected", "selected")
				}
				languagesHTML = append(languagesHTML, elem)
			}
		}
		// i18n end

		fcFlash := GetFailCodeFlash(b.cookieConfig, ctx.W, ctx.R)
		fcText := getFailCodeText(msgr, fcFlash)
		wcFlash := GetWarnCodeFlash(b.cookieConfig, ctx.W, ctx.R)
		wcText := getWarnCodeText(msgr, wcFlash)
		icFlash := GetInfoCodeFlash(b.cookieConfig, ctx.W, ctx.R)
		icText := getInfoCodeText(msgr, icFlash)
		wlFlash := GetWrongLoginInputFlash(b.cookieConfig, ctx.W, ctx.R)

		if icFlash == InfoCodePasswordSuccessfullyChanged {
			wcText = ""
		}

		wrapperClass := "flex pt-8 flex-col max-w-md mx-auto"

		var oauthHTML HTMLComponent
		if b.oauthEnabled {
			ul := Div().Class("flex flex-col justify-center mt-8 text-center")
			for _, provider := range b.providers {
				ul.AppendChildren(
					A().
						Href(fmt.Sprintf("%s?provider=%s", b.oauthBeginURL, provider.Key)).
						Class("px-6 py-3 mt-4 font-semibold text-gray-900 bg-white border-2 border-gray-500 rounded-md shadow outline-none hover:bg-yellow-50 hover:border-yellow-400 focus:outline-none").
						Children(
							provider.Logo,
							Text(provider.Text),
						),
				)
			}

			oauthHTML = Div(
				ul,
			)
		}

		isRecaptchaEnabled := b.recaptchaEnabled

		var userPassHTML HTMLComponent
		if b.userPassEnabled {
			wrapperClass += " pt-16"
			userPassHTML = Div(
				Form(
					Div(
						Label(msgr.AccountLabel).Class("block mb-2 text-sm text-gray-600 dark:text-gray-200").For("account"),
						Input("account").Placeholder(msgr.AccountPlaceholder).Class("block w-full px-4 py-2 mt-2 text-gray-700 placeholder-gray-400 bg-white border border-gray-200 rounded-md dark:placeholder-gray-600 dark:bg-gray-900 dark:text-gray-300 dark:border-gray-700 focus:border-blue-400 dark:focus:border-blue-400 focus:ring-blue-400 focus:outline-none focus:ring focus:ring-opacity-40").
							Value(wlFlash.Account),
					),
					Div(
						Label(msgr.PasswordLabel).Class("block mb-2 text-sm text-gray-600 dark:text-gray-200").For("password"),
						passwordInputWithRevealFunction("password", msgr.PasswordPlaceholder, "password", wlFlash.Password),
					).Class("mt-6"),
					If(isRecaptchaEnabled,
						Div(
							// recaptcha response token
							Input("token").Id("token"),
						).Class("hidden"),
					),
					Div(
						Button(msgr.SignInBtn).Class("w-full px-6 py-3 tracking-wide text-white transition-colors duration-200 transform bg-blue-500 rounded-md hover:bg-blue-400 focus:outline-none focus:bg-blue-400 focus:ring focus:ring-blue-300 focus:ring-opacity-50").
							ClassIf("g-recaptcha", isRecaptchaEnabled).AttrIf("data-sitekey", b.recaptchaConfig.SiteKey, isRecaptchaEnabled).AttrIf("data-callback", "onSubmit", isRecaptchaEnabled),
					).Class("mt-6"),
				).Id("login-form").Method(http.MethodPost).Action(b.userPassLoginURL),
				If(!b.noForgetPasswordLink,
					Div(
						A(Text(msgr.ForgetPasswordLink)).Href(b.userPassForgetPassPageURL).
							Class("text-gray-500"),
					).Class("text-right mt-2"),
				),
			)
		}

		r.PageTitle = "Sign In"
		var bodyForm HTMLComponent
		bodyForm = Div(
			userPassHTML,
			oauthHTML,
			If(len(languagesHTML) > 0,
				Select(
					languagesHTML...,
				).Class("mt-12 bg-gray-50 border border-gray-300 text-gray-900 text-sm rounded-lg focus:ring-blue-500 focus:border-blue-500 block w-full p-2.5 dark:bg-gray-700 dark:border-gray-600 dark:placeholder-gray-400 dark:text-white dark:focus:ring-blue-500 dark:focus:border-blue-500").
					Attr("onChange", "window.location.href=this.value"),
			),
		).Class(wrapperClass)

		if b.defaultLoginPageFormWrapFunc != nil {
			bodyForm = b.defaultLoginPageFormWrapFunc(bodyForm)
		}

		r.Body = Div(
			If(isRecaptchaEnabled,
				Script("").Src("https://www.google.com/recaptcha/api.js"),
				Script(`
function onSubmit(token) {
	document.getElementById("token").value = token;
	document.getElementById("login-form").submit();
}
`)),
			Style(StyleCSS),
			errNotice(fcText),
			warnNotice(wcText),
			infoNotice(icText),
			bodyForm,
		)

		return
	}
}

func defaultForgetPasswordPage(b *Builder) web.PageFunc {
	return func(ctx *web.EventContext) (r web.PageResponse, err error) {
		msgr := i18n.MustGetModuleMessages(ctx.R, I18nLoginKey, Messages_en_US).(*Messages)

		fcFlash := GetFailCodeFlash(b.cookieConfig, ctx.W, ctx.R)
		fcText := getFailCodeText(msgr, fcFlash)
		inputFlash := GetWrongForgetPasswordInputFlash(b.cookieConfig, ctx.W, ctx.R)
		secondsToResend := GetSecondsToRedoFlash(b.cookieConfig, ctx.W, ctx.R)
		activeBtnText := msgr.SendResetPasswordEmailBtn
		activeBtnClass := "w-full px-6 py-3 tracking-wide text-white transition-colors duration-200 transform bg-blue-500 rounded-md hover:bg-blue-400 focus:outline-none focus:bg-blue-400 focus:ring focus:ring-blue-300 focus:ring-opacity-50"
		inactiveBtnText := msgr.ResendResetPasswordEmailBtn
		inactiveBtnClass := "w-full px-6 py-3 tracking-wide text-white transition-colors duration-200 transform bg-gray-500 rounded-md"
		inactiveBtnTextWithInitSeconds := fmt.Sprintf("%s (%d)", inactiveBtnText, secondsToResend)

		doTOTP := ctx.R.URL.Query().Get("totp") == "1"
		actionURL := b.userPassDoSendResetPassLinkURL
		if doTOTP {
			actionURL = mustSetQuery(actionURL, "totp", "1")
		}

		isRecaptchaEnabled := b.recaptchaEnabled

		r.PageTitle = "Forget Your Password?"
		r.Body = Div(
			If(isRecaptchaEnabled,
				Script("").Src("https://www.google.com/recaptcha/api.js"),
				Script(`
function onSubmit(token) {
	document.getElementById("token").value = token;
	document.getElementById("forget-form").submit();
}
`)),
			Style(StyleCSS),
			errNotice(fcText),
			If(secondsToResend > 0,
				warnNotice(msgr.SendEmailTooFrequentlyNotice),
			),
			Div(
				H1(msgr.ForgotMyPasswordTitle).Class("leading-tight text-3xl mt-0 mb-6"),
				Form(
					Div(
						Label(msgr.ForgetPasswordEmailLabel).Class("block mb-2 text-sm text-gray-600 dark:text-gray-200").For("account"),
						Input("account").Placeholder(msgr.ForgetPasswordEmailPlaceholder).Class("block w-full px-4 py-2 mt-2 text-gray-700 placeholder-gray-400 bg-white border border-gray-200 rounded-md dark:placeholder-gray-600 dark:bg-gray-900 dark:text-gray-300 dark:border-gray-700 focus:border-blue-400 dark:focus:border-blue-400 focus:ring-blue-400 focus:outline-none focus:ring focus:ring-opacity-40").Value(inputFlash.Account),
					),
					If(doTOTP,
						Div(
							Label(msgr.TOTPValidateCodeLabel).Class("block mb-2 text-sm text-gray-600 dark:text-gray-200").For("otp"),
							Input("otp").Placeholder(msgr.TOTPValidateCodePlaceholder).
								Class("block w-full px-4 py-2 mt-2 text-gray-700 placeholder-gray-400 bg-white border border-gray-200 rounded-md dark:placeholder-gray-600 dark:bg-gray-900 dark:text-gray-300 dark:border-gray-700 focus:border-blue-400 dark:focus:border-blue-400 focus:ring-blue-400 focus:outline-none focus:ring focus:ring-opacity-40").
								Value(inputFlash.TOTP),
						).Class("mt-6"),
					),
					If(isRecaptchaEnabled,
						Div(
							// recaptcha response token
							Input("token").Id("token"),
						).Class("hidden"),
					),
					Div(
						If(secondsToResend > 0,
							Button(inactiveBtnTextWithInitSeconds).Id("submitBtn").Class(inactiveBtnClass).Disabled(true),
						).Else(
							Button(activeBtnText).Class(activeBtnClass).
								ClassIf("g-recaptcha", isRecaptchaEnabled).AttrIf("data-sitekey", b.recaptchaConfig.SiteKey, isRecaptchaEnabled).AttrIf("data-callback", "onSubmit", isRecaptchaEnabled),
						),
					).Class("mt-6"),
				).Id("forget-form").Method(http.MethodPost).Action(actionURL),
			).Class("flex pt-8 flex-col max-w-md mx-auto pt-16"),
		)

		if secondsToResend > 0 {
			ctx.Injector.TailHTML(fmt.Sprintf(`
<script>
(function(){
    var secondsToResend = %d;
    var btnText = "%s";
    var submitBtn = document.getElementById("submitBtn");
    var interv = setInterval(function(){
        secondsToResend--;
        if (secondsToResend === 0) {
            clearInterval(interv);
            submitBtn.innerText = btnText;
            submitBtn.className = "%s";
            submitBtn.disabled = false;
            return;
        }
        submitBtn.innerText = btnText + " (" + secondsToResend + ")" ;
    }, 1000);
})();
</script>
        `, secondsToResend, inactiveBtnText, activeBtnClass))
		}
		return
	}
}

func defaultResetPasswordLinkSentPage(b *Builder) web.PageFunc {
	return func(ctx *web.EventContext) (r web.PageResponse, err error) {
		msgr := i18n.MustGetModuleMessages(ctx.R, I18nLoginKey, Messages_en_US).(*Messages)

		a := ctx.R.URL.Query().Get("a")

		r.PageTitle = "Forget Your Password?"
		r.Body = Div(
			Style(StyleCSS),
			Div(
				H1(fmt.Sprintf("%s %s.", msgr.ResetPasswordLinkWasSentTo, a)).Class("leading-tight text-2xl mt-0 mb-4"),
				H2(msgr.ResetPasswordLinkSentPrompt).Class("leading-tight text-1xl mt-0"),
			).Class("flex pt-8 flex-col max-w-md mx-auto pt-16"),
		)
		return
	}
}

func defaultResetPasswordPage(b *Builder) web.PageFunc {
	return func(ctx *web.EventContext) (r web.PageResponse, err error) {
		msgr := i18n.MustGetModuleMessages(ctx.R, I18nLoginKey, Messages_en_US).(*Messages)

		fcFlash := GetFailCodeFlash(b.cookieConfig, ctx.W, ctx.R)
		errMsg := getFailCodeText(msgr, fcFlash)
		if errMsg == "" {
			errMsg = GetCustomErrorMessageFlash(b.cookieConfig, ctx.W, ctx.R)
		}
		wrpiFlash := GetWrongResetPasswordInputFlash(b.cookieConfig, ctx.W, ctx.R)

		doTOTP := ctx.R.URL.Query().Get("totp") == "1"
		actionURL := b.userPassDoResetPassURL
		if doTOTP {
			actionURL = mustSetQuery(actionURL, "totp", "1")
		}

		var user interface{}

		r.PageTitle = "Reset Password"

		query := ctx.R.URL.Query()
		id := query.Get("id")
		if id == "" {
			r.Body = Div(Text("user not found"))
			return r, nil
		} else {
			user, err = b.findUserByID(id)
			if err != nil {
				if err == gorm.ErrRecordNotFound {
					r.Body = Div(Text("user not found"))
					return r, nil
				}
				r.Body = Div(Text("system error"))
				return r, nil
			}
		}
		token := query.Get("token")
		if token == "" {
			r.Body = Div(Text("invalid token"))
			return r, nil
		} else {
			storedToken, _, expired := user.(UserPasser).GetResetPasswordToken()
			if expired {
				r.Body = Div(Text("token expired"))
				return r, nil
			}
			if token != storedToken {
				r.Body = Div(Text("invalid token"))
				return r, nil
			}
		}

		r.Body = Div(
			Style(StyleCSS),
			errNotice(errMsg),
			Div(
				H1(msgr.ResetYourPasswordTitle).Class("leading-tight text-3xl mt-0 mb-6"),
				Form(
					Input("user_id").Type("hidden").Value(id),
					Input("token").Type("hidden").Value(token),
					Div(
						Label(msgr.ResetPasswordLabel).Class("block mb-2 text-sm text-gray-600 dark:text-gray-200").For("password"),
						passwordInputWithRevealFunction("password", msgr.ResetPasswordPlaceholder, "password", wrpiFlash.Password),
					),
					Div(
						Label(msgr.ResetPasswordConfirmLabel).Class("block mb-2 text-sm text-gray-600 dark:text-gray-200").For("confirm_password"),
						passwordInputWithRevealFunction("confirm_password", msgr.ResetPasswordConfirmPlaceholder, "confirm_password", wrpiFlash.ConfirmPassword),
					).Class("mt-6"),
					If(doTOTP,
						Div(
							Label(msgr.TOTPValidateCodeLabel).Class("block mb-2 text-sm text-gray-600 dark:text-gray-200").For("otp"),
							Input("otp").Placeholder(msgr.TOTPValidateCodePlaceholder).
								Class("block w-full px-4 py-2 mt-2 text-gray-700 placeholder-gray-400 bg-white border border-gray-200 rounded-md dark:placeholder-gray-600 dark:bg-gray-900 dark:text-gray-300 dark:border-gray-700 focus:border-blue-400 dark:focus:border-blue-400 focus:ring-blue-400 focus:outline-none focus:ring focus:ring-opacity-40").
								Value(wrpiFlash.TOTP),
						).Class("mt-6"),
					),
					Div(
						Button(msgr.Confirm).Class("w-full px-6 py-3 tracking-wide text-white transition-colors duration-200 transform bg-blue-500 rounded-md hover:bg-blue-400 focus:outline-none focus:bg-blue-400 focus:ring focus:ring-blue-300 focus:ring-opacity-50"),
					).Class("mt-6"),
				).Method(http.MethodPost).Action(actionURL),
			).Class("flex pt-8 flex-col max-w-md mx-auto pt-16"),
		)
		return
	}
}

func defaultChangePasswordPage(b *Builder) web.PageFunc {
	return func(ctx *web.EventContext) (r web.PageResponse, err error) {
		msgr := i18n.MustGetModuleMessages(ctx.R, I18nLoginKey, Messages_en_US).(*Messages)

		fcFlash := GetFailCodeFlash(b.cookieConfig, ctx.W, ctx.R)
		errMsg := getFailCodeText(msgr, fcFlash)
		if errMsg == "" {
			errMsg = GetCustomErrorMessageFlash(b.cookieConfig, ctx.W, ctx.R)
		}
		input := GetWrongChangePasswordInputFlash(b.cookieConfig, ctx.W, ctx.R)

		r.PageTitle = "Change Password"

		r.Body = Div(
			Style(StyleCSS),
			errNotice(errMsg),
			Div(
				H1(msgr.ChangePasswordTitle).Class("leading-tight text-3xl mt-0 mb-6"),
				Form(
					Div(
						Label(msgr.ChangePasswordOldLabel).Class("block mb-2 text-sm text-gray-600 dark:text-gray-200").For("old_password"),
						passwordInputWithRevealFunction("old_password", msgr.ChangePasswordOldPlaceholder, "old_password", input.OldPassword),
					),
					Div(
						Label(msgr.ChangePasswordNewLabel).Class("block mb-2 text-sm text-gray-600 dark:text-gray-200").For("password"),
						passwordInputWithRevealFunction("password", msgr.ChangePasswordNewPlaceholder, "password", input.NewPassword),
					).Class("mt-6"),
					Div(
						Label(msgr.ChangePasswordNewConfirmLabel).Class("block mb-2 text-sm text-gray-600 dark:text-gray-200").For("confirm_password"),
						passwordInputWithRevealFunction("confirm_password", msgr.ChangePasswordNewConfirmPlaceholder, "confirm_password", input.ConfirmPassword),
					).Class("mt-6"),
					If(b.totpEnabled,
						Div(
							Label(msgr.TOTPValidateCodeLabel).Class("block mb-2 text-sm text-gray-600 dark:text-gray-200").For("otp"),
							Input("otp").Placeholder(msgr.TOTPValidateCodePlaceholder).
								Class("block w-full px-4 py-2 mt-2 text-gray-700 placeholder-gray-400 bg-white border border-gray-200 rounded-md dark:placeholder-gray-600 dark:bg-gray-900 dark:text-gray-300 dark:border-gray-700 focus:border-blue-400 dark:focus:border-blue-400 focus:ring-blue-400 focus:outline-none focus:ring focus:ring-opacity-40").
								Value(input.TOTP),
						).Class("mt-6"),
					),
					Div(
						Button(msgr.Confirm).Class("w-full px-6 py-3 tracking-wide text-white transition-colors duration-200 transform bg-blue-500 rounded-md hover:bg-blue-400 focus:outline-none focus:bg-blue-400 focus:ring focus:ring-blue-300 focus:ring-opacity-50"),
					).Class("mt-6"),
				).Method(http.MethodPost).Action(b.userPassDoChangePassURL),
			).Class("flex pt-8 flex-col max-w-md mx-auto pt-16"),
		)
		return
	}
}

func defaultTOTPSetupPage(b *Builder) web.PageFunc {
	return func(ctx *web.EventContext) (r web.PageResponse, err error) {
		msgr := i18n.MustGetModuleMessages(ctx.R, I18nLoginKey, Messages_en_US).(*Messages)

		fcFlash := GetFailCodeFlash(b.cookieConfig, ctx.W, ctx.R)
		fcText := getFailCodeText(msgr, fcFlash)

		user := GetCurrentUser(ctx.R)
		u := user.(UserPasser)

		var QRCode bytes.Buffer

		// Generate key from TOTPSecret
		var key *otp.Key
		totpSecret := u.GetTOTPSecret()
		if len(totpSecret) == 0 {
			r.Body = errorBody("need setup totp")
			return
		}
		key, err = otp.NewKeyFromURL(
			fmt.Sprintf(otpKeyFormat,
				url.PathEscape(b.totpIssuer),
				url.PathEscape(u.GetAccountName()),
				url.QueryEscape(b.totpIssuer),
				url.QueryEscape(totpSecret),
			),
		)

		img, err := key.Image(200, 200)
		if err != nil {
			r.Body = errorBody(err.Error())
			return
		}

		err = png.Encode(&QRCode, img)
		if err != nil {
			r.Body = errorBody(err.Error())
			return
		}

		wrapperClass := "flex pt-8 flex-col max-w-md mx-auto relative text-center"
		labelClass := "w-80 text-sm mb-8 font-semibold text-gray-700 tracking-wide"

		r.PageTitle = "TOTP Setup"
		r.Body = Div(
			Style(StyleCSS),
			errNotice(fcText),
			Div(
				Div(
					H1(msgr.TOTPSetupTitle).
						Class("text-3xl font-bold mb-4"),
					Label(msgr.TOTPSetupScanPrompt).
						Class(labelClass),
				),
				Div(
					Img(fmt.Sprintf("data:image/png;base64,%s", base64.StdEncoding.EncodeToString(QRCode.Bytes()))),
				).Class("my-2 flex items-center justify-center"),
				Div(
					Label(msgr.TOTPSetupSecretPrompt).
						Class(labelClass),
				),
				Div(Label(u.GetTOTPSecret()).Class("text-sm font-bold")).Class("my-4"),
				Form(
					Label(msgr.TOTPSetupEnterCodePrompt).Class(labelClass),
					Input("otp").Placeholder(msgr.TOTPSetupCodePlaceholder).
						Class("my-6 block w-full px-4 py-2 text-gray-700 placeholder-gray-400 bg-white border border-gray-200 rounded-md dark:placeholder-gray-600 dark:bg-gray-900 dark:text-gray-300 dark:border-gray-700 focus:border-blue-400 dark:focus:border-blue-400 focus:ring-blue-400 focus:outline-none focus:ring focus:ring-opacity-40"),
					Button(msgr.Verify).
						Class("w-full px-6 py-3 tracking-wide text-white transition-colors duration-200 transform bg-blue-500 rounded-md hover:bg-blue-400 focus:outline-none focus:bg-blue-400 focus:ring focus:ring-blue-300 focus:ring-opacity-50"),
				).Method(http.MethodPost).Action(b.totpDoURL),
			).Class(wrapperClass),
		)

		return
	}
}

func defaultTOTPValidatePage(b *Builder) web.PageFunc {
	return func(ctx *web.EventContext) (r web.PageResponse, err error) {
		msgr := i18n.MustGetModuleMessages(ctx.R, I18nLoginKey, Messages_en_US).(*Messages)

		fcFlash := GetFailCodeFlash(b.cookieConfig, ctx.W, ctx.R)
		fcText := getFailCodeText(msgr, fcFlash)

		wrapperClass := "flex pt-8 flex-col max-w-md mx-auto relative text-center"
		labelClass := "w-80 text-sm mb-8 font-semibold text-gray-700 tracking-wide"

		r.PageTitle = "TOTP Validate"
		r.Body = Div(
			Style(StyleCSS),
			errNotice(fcText),
			Div(
				Div(
					H1(msgr.TOTPValidateTitle).
						Class("text-3xl font-bold mb-4"),
					Label(msgr.TOTPValidateEnterCodePrompt).
						Class(labelClass),
				),
				Form(
					Input("otp").Placeholder(msgr.TOTPValidateCodePlaceholder).
						Class("my-6 block w-full px-4 py-2 text-gray-700 placeholder-gray-400 bg-white border border-gray-200 rounded-md dark:placeholder-gray-600 dark:bg-gray-900 dark:text-gray-300 dark:border-gray-700 focus:border-blue-400 dark:focus:border-blue-400 focus:ring-blue-400 focus:outline-none focus:ring focus:ring-opacity-40").
						Attr("autofocus", true),
					Button(msgr.Verify).
						Class("w-full px-6 py-3 tracking-wide text-white transition-colors duration-200 transform bg-blue-500 rounded-md hover:bg-blue-400 focus:outline-none focus:bg-blue-400 focus:ring focus:ring-blue-300 focus:ring-opacity-50"),
				).Method(http.MethodPost).Action(b.totpDoURL),
			).Class(wrapperClass),
		)

		return
	}
}
