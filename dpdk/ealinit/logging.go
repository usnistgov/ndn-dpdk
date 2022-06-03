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
	"golang.org/x/sys/unix"
)

const (
	logPkgDPDK   = "DPDK"
	logPkgSPDK   = "SPDK"
	logPrefixNDN = "NDN."

	logErrorPrefix = " ERROR={"
	logErrorSuffix = "}"

	reLogDumpID  = 1
	reLogDumpPkg = 2

	reLogLineType    = 1
	reLogLineLevel   = 2
	reLogLineLCore   = 3
	reLogLineIsNDN   = 4
	reLogLineFullMsg = 5
	reLogLineMsg     = 6
	reLogLineKV      = 7
	reLogLineError   = 8

	reErrnoErrno = 1
)

var (
	logTypes  = map[int]string{}
	logStream *cptr.FilePipeCGo

	reLogDump = regexp.MustCompile(`(?m)^id (\d+): ([^,]+), level is `)
	reLogLine = regexp.MustCompile(`^(\d+) (\d) (\d+) \* (NDN: )?((.*?)((?: [^ =]+=[^ =]+)*?)(` + logErrorPrefix + `[^}]+` + logErrorSuffix + `)?)\n`)
	reErrno   = regexp.MustCompile(`^errno<-?(\d+)>$`)
)

func updateLogTypes() {
	data, e := cptr.CaptureFileDump(func(fp unsafe.Pointer) { C.rte_log_dump((*C.FILE)(fp)) })
	if e != nil {
		logger.Error("rte_log_dump", zap.Error(e))
		return
	}

	for _, m := range reLogDump.FindAllSubmatch(data, -1) {
		id, e := strconv.Atoi(string(m[reLogDumpID]))
		if e != nil {
			continue
		}
		pkg := string(m[reLogDumpPkg])
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
	for id, logType := range logTypes {
		pl := logging.GetLevel(logType)
		idC := C.uint32_t(id)
		set := func() { C.rte_log_set_level(idC, parseLogLevel(pl.Level())) }
		pl.SetCallback(set)
		set()
	}
}

func init() {
	updateLogLevels()
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
		logger.Error("Logger_Dpdk_Init", zap.Error(eal.MakeErrno(res)))
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

	logTypeID, _ := strconv.Atoi(string(m[reLogLineType]))
	var l *zap.Logger
	if logName, ok := logTypes[logTypeID]; ok {
		l = logging.Named(logName)
	} else {
		l = logging.Named(logPkgDPDK)
	}
	if lc, _ := strconv.Atoi(string(m[reLogLineLCore])); lc != math.MaxUint32 {
		l = l.Named(string(m[reLogLineLCore]))
	}

	lvl := dpdk2zapLogLevels[m[reLogLineLevel][0]]
	if len(m[reLogLineIsNDN]) == 0 {
		l.Check(lvl, string(m[reLogLineFullMsg])).Write()
		return
	}

	msg := string(m[reLogLineMsg])
	ce := l.Check(lvl, msg)
	if ce == nil {
		return
	}

	pairs := bytes.Split(m[reLogLineKV], []byte{' '})[1:]
	fields := make([]zapcore.Field, 0, len(pairs)+2)

	for _, pair := range pairs {
		kv := bytes.Split(pair, []byte{'='})
		fields = append(fields, zap.ByteString(string(kv[0]), kv[1]))
	}

	if len(m[reLogLineError]) > 0 {
		e := string(m[reLogLineError][len(logErrorPrefix) : len(m[reLogLineError])-len(logErrorSuffix)])
		if e == "-" {
			fields = append(fields, zap.Error(errors.New(msg)))
		} else if em := reErrno.FindStringSubmatch(e); em != nil {
			errno, _ := strconv.ParseUint(em[reErrnoErrno], 10, 64)
			err := unix.Errno(errno)
			fields = append(fields,
				zap.Uint64("errno", errno),
				zap.String("errname", unix.ErrnoName(err)),
				zap.Error(err),
			)
		} else {
			fields = append(fields, zap.Error(errors.New(e)))
		}
	}

	ce.Write(fields...)
}
