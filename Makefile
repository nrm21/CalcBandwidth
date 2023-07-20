NAME        := CalcBandwidth.exe
OUTPUT_BIN  := .\bin\${NAME}
SOURCE      := .\cmd
GO_FLAGS    :=
GO_TAGS     := walk_use_cgo
CGO_ENABLED := 1
LD_FLAGS    := "-w -s -H=windowsgui"

default: move build

move:
	@echo "Moving output..."
	@IF EXIST ${OUTPUT_BIN} ( @move /y ${OUTPUT_BIN} ${OUTPUT_BIN}.bak )

build:
	@echo "Building from source..."
	@set CGO_ENABLED=${CGO_ENABLED}
	./assets/rsrc.exe -manifest=assets/test.manifest -ico=assets/calc.ico -o=cmd/rsrc.syso
	go build -v -tags ${GO_TAGS} ${GO_FLAGS} -ldflags=${LD_FLAGS} -o ${OUTPUT_BIN} ${SOURCE}
