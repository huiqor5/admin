package publish

type Version struct {
	Version       string `gorm:"primary_key;size:128"`
	VersionName   string
	ParentVersion string
	OnlineVersion bool `gorm:"index;default:false"`
}

func (version Version) GetVersion() string {
	return version.Version
}

func (version *Version) SetVersion(v string) {
	version.Version = v
}

func (version Version) GetVersionName() string {
	return version.VersionName
}

func (version *Version) SetVersionName(v string) {
	version.VersionName = v
}
