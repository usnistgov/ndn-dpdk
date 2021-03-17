package ealinit

/*
#include "../../csrc/core/logger.h"
*/
import "C"
import (
	"bufio"
	"bytes"
	"errors"
	"math"
	"regexp"
	"strconv"
	"strings"
	"unsafe"

	"github.com/usnistgov/ndn-dpdk/core/cptr"
	"github.com/usnistgov/ndn-dpdk/core/logging"
	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const (
	logPkgDPDK = "DPDK"
	logPkgSPDK = "SPDK"
	logPkgNDN  = "NDN"
)

var (
	logTypes  = make(map[int]string)
	logStream *cptr.FilePipeCGo

	reLogDump = regexp.MustCompile(`(?m)^id (\d+): ([^,]+), level is `)
	reLogLine = regexp.MustCompile(`^(\d+) (\d) (\d+) \* (?:NDN: )?(.*?)((?: [^ =]+=[^ =]+)*?)( ERROR={[^=}]+})?\n`)
)

func updateLogTypes() {
	data, e := cptr.CaptureFileDump(func(fp unsafe.Pointer) { C.rte_log_dump((*C.FILE)(fp)) })
	if e != nil {
		logger.Error("rte_log_dump",
			zap.Error(e),
		)
		return
	}

	for _, m := range reLogDump.FindAllSubmatch(data, -1) {
		id, e := strconv.Atoi(string(m[1]))
		if e != nil {
			continue
		}
		pkg := string(m[2])
		if strings.HasPrefix(pkg, logPkgNDN+".") {
			pkg = strings.TrimPrefix(pkg, logPkgNDN+".")
		} else if pkg != logPkgSPDK {
			pkg = logPkgDPDK
		}
		logTypes[id] = pkg
	}
}

func updateLogLevels() {
	updateLogTypes()
	for id, logtype := range logTypes {
		C.rte_log_set_level(C.uint32_t(id), parseLogLevel(logging.GetLevel(logtype)))
	}
}

func parseLogLevel(lvl rune) C.uint32_t {
	switch lvl {
	case 'V':
		return C.RTE_LOG_DEBUG
	case 'D':
		return C.RTE_LOG_INFO
	case 'I':
		return C.RTE_LOG_NOTICE
	case 'W':
		return C.RTE_LOG_WARNING
	case 'E':
		return C.RTE_LOG_ERR
	case 'F':
		return C.RTE_LOG_CRIT
	case 'N':
		return C.RTE_LOG_ALERT
	}
	return C.RTE_LOG_NOTICE
}

func initLogStream() {
	var e error
	logStream, e = cptr.NewFilePipeCGo(cptr.FilePipeConfig{NonBlock: true})
	if e != nil {
		logger.Error("cptr.NewFilePipeCGo",
			zap.Error(e),
		)
		return
	}

	res := C.Logger_Dpdk_Init((*C.FILE)(logStream.Writer))
	if res != 0 {
		logger.Error("Logger_Dpdk_Init",
			zap.Error(eal.Errno(-res)),
		)
		return
	}

	go processLogStream()
}

var dpdk2zapLogLevels = map[byte]zapcore.Level{
	C.RTE_LOG_EMERG:   zapcore.ErrorLevel,
	C.RTE_LOG_ALERT:   zapcore.ErrorLevel,
	C.RTE_LOG_CRIT:    zapcore.ErrorLevel,
	C.RTE_LOG_ERR:     zapcore.ErrorLevel,
	C.RTE_LOG_WARNING: zapcore.WarnLevel,
	C.RTE_LOG_NOTICE:  zapcore.InfoLevel,
	C.RTE_LOG_INFO:    zapcore.DebugLevel,
	C.RTE_LOG_DEBUG:   zapcore.DebugLevel,
}

func processLogStream() {
	r := bufio.NewReader(logStream.Reader)
	for {
		line, e := r.ReadBytes('\n')
		if e != nil {
			logger.Error("logStream.Reader read line error",
				zap.Error(e),
			)
		}

		processLogLine(line)
	}
}

func processLogLine(line []byte) {
	m := reLogLine.FindSubmatch(line)
	if m == nil {
		return
	}

	logtype, _ := strconv.Atoi(string(m[1]))
	l := logging.Named(logTypes[logtype])
	lvl := dpdk2zapLogLevels[m[2][0]]
	msg := string(m[4])
	ce := l.Check(lvl, msg)
	if ce == nil {
		return
	}

	lc, _ := strconv.Atoi(string(m[3]))

	pairs := bytes.Split(m[5], []byte{' '})[1:]
	fields := make([]zapcore.Field, 0, len(pairs)+2)

	for _, pair := range pairs {
		kv := bytes.Split(pair, []byte{'='})
		fields = append(fields, zap.ByteString(string(kv[0]), kv[1]))
	}

	if lc != math.MaxUint32 {
		fields = append(fields, zap.Int("lc", lc))
	}

	if len(m[6]) > 0 {
		e := string(m[6])
		if e == "-" {
			fields = append(fields, zap.Error(errors.New(msg)))
		} else {
			fields = append(fields, zap.Error(errors.New(e)))
		}
	}

	ce.Write(fields...)
}
