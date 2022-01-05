go build -tags walk_use_cgo -ldflags="-H windowsgui" -o "./bin/CalcBandwidth.new" ./src
move /y .\bin\CalcBandwidth.exe .\bin\CalcBandwidth.exe.bak
move /y .\bin\CalcBandwidth.new .\bin\CalcBandwidth.exe