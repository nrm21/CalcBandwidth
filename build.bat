@rem No longer necessary as the build tasks should do this exact job

go build -tags walk_use_cgo -ldflags="-w -s -H=windowsgui" -o "./bin/CalcBandwidth.new" ./src
move /y .\bin\CalcBandwidth.exe .\bin\CalcBandwidth.exe.bak
move /y .\bin\CalcBandwidth.new .\bin\CalcBandwidth.exe