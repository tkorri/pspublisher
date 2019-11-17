pspublisher
===========

pspublisher is a simple command line tool for publishing apk files to Google
Play Store.

Currently the tool supports apk upload, release notes and ProGuard mapping
files.
    
## Usage

    Usage: ./pspublisher <command> [<args>]
    Supported commands
        uploadApk	Upload Apk to Play Store

### uploadApk command help

    Usage: ./pspublisher uploadApk [<args>]
    Supported arguments
      -apk string
            Required. Path to apk file to upload
      -debug
            Optional. Enable debug logging
      -id string
            Required. Application package name id
      -key string
            Required. Service account key file
      -mapping string
            Optional. Path to ProGuard mapping file to upload
      -releasenotes string
            Optional. Release notes (default "Uploaded with pspublisher")
      -releasenotesfile string
            Optional. Path to file containing release notes
      -status string
            Optional. Publish status (default "completed")
      -track string
            Required. Name of track where to publish the apk
      -verbose
            Optional. Enable verbose logging

### Example call

    pspublisher uploadApk
                -id net.example.application \
	            -key /path/to/keys.json \
	            -track internal \
                -apk /path/to/apk
