NAME        ?= CalcBandwidth.exe
OUTPUT_BIN  ?= .\bin\${NAME}
SOURCE      ?= .\src
GO_FLAGS    ?=
GO_TAGS     ?= walk_use_cgo
CGO_ENABLED ?= 0
LD_FLAGS    ?= "-w -s -H=windowsgui"

default:
	@move /y ${OUTPUT_BIN} ${OUTPUT_BIN}.bak
	@make build

build:
	@set CGO_ENABLED=${CGO_ENABLED}
	go build -tags ${GO_TAGS} ${GO_FLAGS} -ldflags=${LD_FLAGS} -o ${OUTPUT_BIN} ${SOURCE}
