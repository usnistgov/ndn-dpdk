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
	_ "github.com/usnistgov/ndn-dpdk/core/logging/logginggql"
	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const (
	logPkgDPDK   = "DPDK"
	logPkgSPDK   = "SPDK"
	logPrefixNDN = "NDN."

	logErrorPrefix = " ERROR={"
	logErrorSuffix = "}"
)

var (
	logTypes  = make(map[int]string)
	logStream *cptr.FilePipeCGo

	reLogDump = regexp.MustCompile(`(?m)^id (\d+): ([^,]+), level is `)
	reLogLine = regexp.MustCompile(`^(\d+) (\d) (\d+) \* (?:NDN: )?(.*?)((?: [^ =]+=[^ =]+)*?)(` + logErrorPrefix + `[^}]+` + logErrorSuffix + `)?\n`)
)

func updateLogTypes() {
	data, e := cptr.CaptureFileDump(func(fp unsafe.Pointer) { C.rte_log_dump((*C.FILE)(fp)) })
	if e != nil {
		logger.Error("rte_log_dump", zap.Error(e))
		return
	}

	for _, m := range reLogDump.FindAllSubmatch(data, -1) {
		id, e := strconv.Atoi(string(m[1]))
		if e != nil {
			continue
		}
		pkg := string(m[2])
		if strings.HasPrefix(pkg, logPrefixNDN) {
			pkg = pkg[len(logPrefixNDN):]
		} else if pkg != logPkgSPDK {
			pkg = logPkgDPDK
		}
		logTypes[id] = pkg
	}
}

func updateLogLevels() {
	updateLogTypes()
	for id, logtype := range logTypes {
		pl := logging.GetLevel(logtype)
		idC := C.uint32_t(id)
		set := func() { C.rte_log_set_level(idC, parseLogLevel(pl.Level())) }
		pl.SetCallback(set)
		set()
	}
}

func parseLogLevel(lvl byte) C.uint32_t {
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
		logger.Error("cptr.NewFilePipeCGo", zap.Error(e))
		return
	}

	res := C.Logger_Dpdk_Init((*C.FILE)(logStream.Writer))
	if res != 0 {
		logger.Error("Logger_Dpdk_Init", zap.Error(eal.Errno(-res)))
		return
	}

	go processLogStream()
}

var dpdk2zapLogLevels = map[byte]zapcore.Level{
	'0' + C.RTE_LOG_EMERG:   zapcore.ErrorLevel,
	'0' + C.RTE_LOG_ALERT:   zapcore.ErrorLevel,
	'0' + C.RTE_LOG_CRIT:    zapcore.ErrorLevel,
	'0' + C.RTE_LOG_ERR:     zapcore.ErrorLevel,
	'0' + C.RTE_LOG_WARNING: zapcore.WarnLevel,
	'0' + C.RTE_LOG_NOTICE:  zapcore.InfoLevel,
	'0' + C.RTE_LOG_INFO:    zapcore.DebugLevel,
	'0' + C.RTE_LOG_DEBUG:   zapcore.DebugLevel,
}

func processLogStream() {
	r := bufio.NewReader(logStream.Reader)
	for {
		line, e := r.ReadBytes('\n')
		if e != nil {
			logger.Error("logStream.Reader read line error", zap.Error(e))
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
	logName := logTypes[logtype]
	var l *zap.Logger
	if logName == "" {
		l = logging.Named(logPkgDPDK)
	} else {
		l = logging.Named(logTypes[logtype])
	}
	if lc, _ := strconv.Atoi(string(m[3])); lc != math.MaxUint32 {
		l = l.Named(string(m[3]))
	}

	lvl := dpdk2zapLogLevels[m[2][0]]
	msg := string(m[4])
	ce := l.Check(lvl, msg)
	if ce == nil {
		return
	}

	pairs := bytes.Split(m[5], []byte{' '})[1:]
	fields := make([]zapcore.Field, 0, len(pairs)+2)

	for _, pair := range pairs {
		kv := bytes.Split(pair, []byte{'='})
		fields = append(fields, zap.ByteString(string(kv[0]), kv[1]))
	}

	if len(m[6]) > 0 {
		e := string(m[6][len(logErrorPrefix) : len(m[6])-len(logErrorSuffix)])
		if e == "-" {
			fields = append(fields, zap.Error(errors.New(msg)))
		} else {
			fields = append(fields, zap.Error(errors.New(e)))
		}
	}

	ce.Write(fields...)
}
