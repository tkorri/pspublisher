package main

import (
	"context"
	"flag"
	"github.com/tkorri/pspublisher/command"
	"github.com/tkorri/pspublisher/logger"
	"google.golang.org/api/androidpublisher/v3"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/option"
	"io/ioutil"
	"os"
	"path/filepath"
)

const versionString = "1.0.0"

const uploadApkCmdString = "uploadApk"

const (
	Id               = "id"
	Key              = "key"
	Apk              = "apk"
	Track            = "track"
	Mapping          = "mapping"
	ReleaseNotes     = "releasenotes"
	ReleaseNotesFile = "releasenotesfile"
	Status           = "status"
	Verbose          = "verbose"
	Debug            = "debug"
)

const (
	PublishStatusCompleted  string = "completed"
	PublishStatusDraft      string = "draft"
	PublishStatusHalted     string = "halted"
	PublishStatusInProgress string = "inProgress"
)

var log *logger.Logger

func main() {
	flag.Usage = showHelp
	flag.CommandLine.SetOutput(os.Stderr)

	// Upload apk
	uploadCommand := command.New(uploadApkCmdString)
	uploadCommand.AddString(Id, "", "Required. Application package name id")
	uploadCommand.AddString(Key, "", "Required. Service account key file")
	uploadCommand.AddString(Apk, "", "Required. Path to apk file to upload")
	uploadCommand.AddString(Track, "", "Required. Name of track where to publish the apk")
	uploadCommand.AddString(Mapping, "", "Optional. Path to ProGuard mapping file to upload")
	uploadCommand.AddString(ReleaseNotes, "Uploaded with pspublisher", "Optional. Release notes")
	uploadCommand.AddString(ReleaseNotesFile, "", "Optional. Path to file containing release notes")
	uploadCommand.AddString(Status, PublishStatusCompleted, "Optional. Publish status")
	uploadCommand.AddBool(Verbose, false, "Optional. Enable verbose logging")
	uploadCommand.AddBool(Debug, false, "Optional. Enable debug logging")

	if len(os.Args) < 2 {
		showHelp()
		os.Exit(1)
	}

	switch os.Args[1] {
	case uploadCommand.Name:
		uploadApkCommand(uploadCommand)
	default:
		showHelp()
		os.Exit(1)
	}
}

func showHelp() {
	logger.Errorln("pspublisher %s", versionString)
	logger.Errorln("Usage: %s <command> [<args>]", os.Args[0])
	logger.Errorln("Supported commands")
	logger.Errorln("    %s\tUpload Apk to Play Store", uploadApkCmdString)
}

func showCommandHelp(command *command.Command) {
	logger.Errorln("pspublisher %s", versionString)
	logger.Errorln("Usage: %s %s [<args>]", os.Args[0], command.Name)
	logger.Errorln("Supported arguments")
	command.Command.PrintDefaults()
}

func uploadApkCommand(upload *command.Command) {
	err := upload.Command.Parse(os.Args[2:])
	if err != nil {
		logger.Errorln("Unrecognized parameters:\n%s", err)
		showCommandHelp(upload)
		os.Exit(1)
	}

	log = logger.New(upload.GetBool(Verbose), upload.GetBool(Verbose) || upload.GetBool(Debug))

	if len(os.Args[2:]) == 0 {
		showCommandHelp(upload)
		os.Exit(1)
	}

	if upload.GetString(Id) == "" {
		log.E("Package id is required")
		os.Exit(1)
	}

	if upload.GetString(Key) == "" {
		log.E("Key file is required")
		os.Exit(1)
	}

	if upload.GetString(Track) == "" {
		log.E("Publish track is required")
		os.Exit(1)
	}

	// Check that apk file is available
	apkFile, err := os.Open(upload.GetString(Apk))
	if err != nil {
		log.E("Cannot open apk file:\n%s", err)
		os.Exit(1)
	}
	defer apkFile.Close()

	// Check that mapping file is available if given
	var mappingFile *os.File = nil
	if upload.GetString(Mapping) != "" {
		mappingFile, err = os.Open(upload.GetString(Mapping))
		if err != nil {
			log.E("Cannot open mapping file:\n%s", err)
			os.Exit(1)
		}
		defer mappingFile.Close()
	}

	// Check that upload key file is available
	_, err = os.Open(upload.GetString(Key))
	if err != nil {
		log.E("Cannot open key file file:\n%s", err)
		os.Exit(1)
	}
	defer apkFile.Close()

	// Setup release notes
	var releaseNotes = upload.GetString(ReleaseNotes)
	if upload.GetString(ReleaseNotesFile) != "" {
		notes, err := ioutil.ReadFile(upload.GetString(ReleaseNotesFile))
		if err != nil {
			log.E("Cannot read release notes file contents:\n%s", err)
			os.Exit(1)
		}
		releaseNotes = string(notes)
	}
	// Check that release notes are 500 characters or less
	if len(releaseNotes) > 500 {
		log.E("Release notes cannot exceed 500 characters\n")
		os.Exit(1)
	}

	// Check publish status
	publishStatus := upload.GetString(Status)
	if publishStatus == "" || (publishStatus != PublishStatusCompleted && publishStatus != PublishStatusDraft && publishStatus != PublishStatusInProgress && publishStatus != PublishStatusHalted) {
		log.E("Publish status not recognized. It needs to one of %s, %s, %s, %s", PublishStatusDraft, PublishStatusHalted, PublishStatusInProgress, PublishStatusCompleted)
		os.Exit(1)
	}

	log.I("Creating new service...")
	publisher, err := androidpublisher.NewService(context.Background(), option.WithCredentialsFile(upload.GetString(Key)))
	if err != nil {
		log.E("Failed to create publisher service:\n%s", err)
		os.Exit(1)
	}
	log.I("Service OK")

	log.I("Creating new edit...")
	edit, err := newEdit(publisher, upload.GetString(Id))
	if err != nil {
		log.E("Failed to insert edit:\n%s", err)
		os.Exit(1)
	}
	log.I("Edit %s OK", edit.Id)

	log.I("Getting app listings...")
	listings, err := edit.getListings()
	if err != nil {
		log.E("Failed to list application listings:\n%s", err)
		edit.delete()
		os.Exit(1)
	}
	log.I("Listings OK")

	log.I("Uploading APK...")
	apk, err := edit.uploadApk(apkFile)
	if err != nil {
		log.E("Failed to upload apk:\n%s", err)
		edit.delete()
		os.Exit(1)
	}
	log.I("APK upload %s OK", apk.Binary.Sha1)

	// If mapping file is set and available then proceed with mapping upload
	if mappingFile != nil {
		log.I("Uploading mapping file...")
		_, err = edit.uploadMappingFile(apk.VersionCode, mappingFile)
		if err != nil {
			log.E("Failed to upload mapping file:\n%s", err)
			edit.delete()
			os.Exit(1)
		}
		log.I("Mapping upload OK")
	}

	log.I("Update track...")
	localizedNotes := &androidpublisher.LocalizedText{
		Language: listings.Listings[0].Language,
		Text:     releaseNotes,
	}
	publishTrackRelease := &androidpublisher.TrackRelease{
		ReleaseNotes: []*androidpublisher.LocalizedText{localizedNotes},
		Status:       publishStatus,
		VersionCodes: googleapi.Int64s{apk.VersionCode},
	}
	publishTrack := &androidpublisher.Track{
		Releases: []*androidpublisher.TrackRelease{publishTrackRelease},
	}

	track, err := edit.updateTrack(upload.GetString(Track), publishTrack)
	if err != nil {
		log.E("Failed to update track:\n%s", err)
		edit.delete()
		os.Exit(1)
	}
	log.I("Track %s update OK", track.Track)

	log.I("Validate changes...")
	_, err = edit.validate()
	if err != nil {
		log.E("Failed to validate:\n%s", err)
		edit.delete()
		os.Exit(1)
	}
	log.I("Validation OK")

	log.I("Committing changes...")
	_, err = edit.commit()
	if err != nil {
		log.E("Failed to commit changes:\n%s", err)
		edit.delete()
		os.Exit(1)
	}
	log.I("Commit OK")
}

type Edit struct {
	Service   *androidpublisher.Service
	PackageId string
	Id        string
}

func (e *Edit) delete() {
	log.I("Deleting edit %s...", e.Id)
	deleteCall := e.Service.Edits.Delete(e.PackageId, e.Id)
	err := deleteCall.Do()
	if err != nil {
		log.E("Failed to delete edit:\n%s", err)
		os.Exit(1)
	}
	log.I("Delete %s OK", e.Id)
}

func (e *Edit) getListings() (*androidpublisher.ListingsListResponse, error) {
	log.D("Fetching listings")
	editTracks := e.Service.Edits.Listings.List(e.PackageId, e.Id)
	response, err := editTracks.Do()
	if err != nil {
		return nil, err
	}
	log.V("<-- %+v", response)
	return response, nil
}

func (e *Edit) getTracks() (*androidpublisher.TracksListResponse, error) {
	log.D("Fetching tracks")
	editTracks := e.Service.Edits.Tracks.List(e.PackageId, e.Id)
	response, err := editTracks.Do()
	if err != nil {
		return nil, err
	}
	log.V("<-- %+v", response)
	return response, nil
}

func (e *Edit) uploadApk(apkFile *os.File) (*androidpublisher.Apk, error) {
	log.D("Uploading apk file %s", filepath.Base(apkFile.Name()))
	apkUploadCall := e.Service.Edits.Apks.Upload(e.PackageId, e.Id)
	apkUploadCall.Media(apkFile, googleapi.ContentType("application/vnd.android.package-archive"))
	response, err := apkUploadCall.Do()
	if err != nil {
		return nil, err
	}
	log.V("<-- %+v", response)
	return response, nil
}

func (e *Edit) uploadMappingFile(versionCode int64, mappingFile *os.File) (*androidpublisher.DeobfuscationFilesUploadResponse, error) {
	deObfuscationCall := e.Service.Edits.Deobfuscationfiles.Upload(e.PackageId, e.Id, versionCode, "proguard")
	deObfuscationCall.Media(mappingFile, googleapi.ContentType("application/octet-stream"))
	response, err := deObfuscationCall.Do()
	if err != nil {
		return nil, err
	}
	log.V("<-- %+v", response)
	return response, nil
}

func (e *Edit) updateTrack(trackName string, track *androidpublisher.Track) (*androidpublisher.Track, error) {
	updateCall := e.Service.Edits.Tracks.Update(e.PackageId, e.Id, trackName, track)
	response, err := updateCall.Do()
	if err != nil {
		return nil, err
	}
	log.V("<-- %+v", response)
	return response, nil
}

func (e *Edit) validate() (*androidpublisher.AppEdit, error) {
	validateCall := e.Service.Edits.Validate(e.PackageId, e.Id)
	response, err := validateCall.Do()
	if err != nil {
		return nil, err
	}
	log.V("<-- %+v", response)
	return response, nil
}

func (e *Edit) commit() (*androidpublisher.AppEdit, error) {
	validateCall := e.Service.Edits.Commit(e.PackageId, e.Id)
	response, err := validateCall.Do()
	if err != nil {
		return nil, err
	}
	log.V("<-- %+v", response)
	return response, nil
}

func newEdit(service *androidpublisher.Service, packageId string) (*Edit, error) {
	editService := service.Edits.Insert(packageId, &androidpublisher.AppEdit{})
	appEdit, err := editService.Do()
	if err != nil {
		return nil, err
	}
	log.V("<-- %+v", appEdit)

	return &Edit{
		Service:   service,
		PackageId: packageId,
		Id:        appEdit.Id,
	}, nil
}
