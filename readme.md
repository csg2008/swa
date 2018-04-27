#build
windres -o swa.syso ./res/swa.rc
go build -ldflags="-H windowsgui -linkmode internal  -w"
GOARCH=386 go build -ldflags="-H windowsgui -linkmode internal  -w"