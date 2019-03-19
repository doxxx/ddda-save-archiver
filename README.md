# Dragon's Dogma Dark Arisen Save Archiver

This is a Windows GUI application which monitors a DDDA save file, makes timestamped backups whenever it changes, and allows restoring a particular backup over the primary save file.

It discovers the DDDA save directory by querying the Windows Registry for your Steam folder, and then searches each userdata folder for the DDDA save game folder. If multiple are found, you can select the right one from a dropdown.

## Building

This project uses Go 1.11 modules and should be built outside of the GOPATH. You can build the application with:

```
go build -ldflags="-H windowsgui"
```

This will produce a `ddda-save-archiver.exe` in the current directory.

If you want to enable console output, for debugging perhaps, build it with:

```
go build
```

If you modify the `app.manifest`, you must rebuild it using the `rsrc` tool (`go get github.com/akavel/rsrc`):

```
rsrc -manifest app.manifest -o rsrc.syso
```
