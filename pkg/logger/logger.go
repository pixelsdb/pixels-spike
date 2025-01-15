/*
Copyright 2024 PixelsDB.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package logger

import (
	"github.com/AgentGuo/spike/cmd/server/config"
	"github.com/sirupsen/logrus"
	"io"
	"os"
	"sync"
	"time"
)

var (
	log          *logrus.Logger
	logOnce      sync.Once
	logStringMap = map[string]logrus.Level{
		"info":  logrus.InfoLevel,
		"debug": logrus.DebugLevel,
		"warn":  logrus.WarnLevel,
		"error": logrus.ErrorLevel,
		"fatal": logrus.FatalLevel,
		"panic": logrus.PanicLevel,
	}
)

func GetLogger() *logrus.Logger {
	logOnce.Do(func() {
		log = logrus.New()
		if lvl, ok := logStringMap[config.GetConfig().ServerConfig.LogLevel]; ok {
			log.SetLevel(lvl)
		} else {
			log.SetLevel(logrus.InfoLevel)
		}
		log.SetFormatter(&logrus.TextFormatter{
			FullTimestamp:   true,                  // 显示完整时间戳
			TimestampFormat: "2006-01-02 15:04:05", // 自定义时间格式
			DisableColors:   true,                  // 禁用颜色（适合日志文件）
		})
		//log.SetFormatter(&logrus.JSONFormatter{
		//	TimestampFormat: "2006-01-02 15:04:05", // 自定义时间格式
		//})
		//log.SetReportCaller(true)

		var writerList []io.Writer
		if config.GetConfig().ServerConfig.LogToFile {
			currentTime := time.Now().Format("2006-01-02_15-04-05")
			logFileName := currentTime + ".log"
			// 创建日志文件
			file, err := os.OpenFile(logFileName, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
			writerList = append(writerList, file)
			if err != nil {
				log.Warnf("Failed to log to file: err: %v", err)
			}
		}
		if config.GetConfig().ServerConfig.LogToStd {
			writerList = append(writerList, os.Stdout)
		}
		mw := io.MultiWriter(writerList...)
		log.SetOutput(mw)
	})
	return log
}
