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

package storage

import (
	"pixelsdb.io/spike/cmd/server/config"
	"pixelsdb.io/spike/pkg/logger"
	"pixelsdb.io/spike/pkg/storage/model"
	"github.com/sirupsen/logrus"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"sync"
)

// sudo docker run --name spike-mysql -e MYSQL_ROOT_PASSWORD=spikepassword -p 3306:3306 -d mysql:8.0.31
type Mysql struct {
	db     *gorm.DB
	logger *logrus.Logger
}

var (
	mysqlInitOnce sync.Once
)

func NewMysql() *Mysql {
	var initErr error
	mysqlInitOnce.Do(func() {
		initErr = initMysql(config.GetConfig().ServerConfig.MysqlDsn)
	})
	if initErr != nil {
		logger.GetLogger().Fatal(initErr)
	}

	db, err := gorm.Open(mysql.Open(config.GetConfig().ServerConfig.MysqlDsn), &gorm.Config{})
	if err != nil {
		logger.GetLogger().Fatal(err)
	}
	return &Mysql{
		db:     db,
		logger: logger.GetLogger(),
	}
}

func initMysql(dsn string) error {
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		return err
	}

	// 自动迁移模式
	if err := db.AutoMigrate(&model.FuncMetaData{}); err != nil {
		return err
	}
	if err := db.AutoMigrate(&model.FuncInstance{}); err != nil {
		return err
	}
	if err := db.AutoMigrate(&model.AwsTaskDef{}); err != nil {
		return err
	}
	if err := db.AutoMigrate(&model.ReqScheduleInfo{}); err != nil {
		return err
	}

	return nil
}

func (m *Mysql) Close() error {
	sqlDB, err := m.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

// func metadata

func (m *Mysql) CreateFuncMetaData(data *model.FuncMetaData) error {
	return m.db.Create(data).Error
}

func (m *Mysql) HasFuncMetaDataByFunctionName(functionName string) (bool, error) {
	var data []model.FuncMetaData
	err := m.db.Where(map[string]interface{}{"function_name": functionName}).Find(&data).Error
	if err != nil {
		return false, err
	} else if len(data) == 0 {
		return false, nil
	} else {
		return true, nil
	}
}

func (m *Mysql) GetFuncMetaDataByFunctionName(functionName string) (*model.FuncMetaData, error) {
	data := &model.FuncMetaData{}
	return data, m.db.Where(map[string]interface{}{"function_name": functionName}).First(data).Error
}

func (m *Mysql) GetFuncMetaDataByCondition(condition map[string]interface{}) ([]model.FuncMetaData, error) {
	var data []model.FuncMetaData
	return data, m.db.Where(condition).Find(&data).Error
}

func (m *Mysql) GetFuncMetaData(id uint, data *model.FuncMetaData) error {
	return m.db.First(data, id).Error
}

func (m *Mysql) UpdateFuncMetaData(data *model.FuncMetaData) error {
	return m.db.Save(data).Error
}

func (m *Mysql) DeleteFuncMetaDataByFunctionName(functionName string) error {
	return m.db.Unscoped().Where(map[string]interface{}{"function_name": functionName}).Delete(&model.FuncMetaData{}).Error
}

func (m *Mysql) DeleteFuncMetaData(id uint) error {
	return m.db.Unscoped().Delete(&model.FuncMetaData{}, id).Error
}

// func task data

func (m *Mysql) CreateFuncInstance(data *model.FuncInstance) error {
	return m.db.Create(data).Error
}

func (m *Mysql) GetFuncInstanceByCondition(condition map[string]interface{}) ([]model.FuncInstance, error) {
	var data []model.FuncInstance
	err := m.db.Where(condition).Find(&data).Error
	return data, err
}

func (m *Mysql) GetFuncInstanceByFunctionName(functionName string) ([]model.FuncInstance, error) {
	return m.GetFuncInstanceByCondition(map[string]interface{}{"function_name": functionName})
}

func (m *Mysql) GetFuncInstanceByServiceName(serviceName string) ([]model.FuncInstance, error) {
	return m.GetFuncInstanceByCondition(map[string]interface{}{"service_name": serviceName})
}

func (m *Mysql) GetFuncInstanceByTaskArn(taskArn string) ([]model.FuncInstance, error) {
	return m.GetFuncInstanceByCondition(map[string]interface{}{"task_arn": taskArn})
}

func (m *Mysql) UpdateFuncInstance(data *model.FuncInstance) error {
	return m.db.Save(data).Error
}

func (m *Mysql) UpdateFuncInstanceBatch(data []model.FuncInstance) error {
	tx := m.db.Begin()
	for _, instance := range data {
		if err := tx.Clauses(clause.OnConflict{
			UpdateAll: true,
		}).Save(&instance).Error; err != nil {
			tx.Rollback()
			return err
		}
	}
	return tx.Commit().Error
}

func (m *Mysql) DeleteFuncInstanceServiceName(serviceName string) error {
	return m.db.Unscoped().Where(map[string]interface{}{"aws_service_name": serviceName}).Delete(&model.FuncInstance{}).Error
}

func (m *Mysql) DeleteFuncInstanceFunctionName(functionName string) error {
	return m.db.Unscoped().Where(map[string]interface{}{"function_name": functionName}).Delete(&model.FuncInstance{}).Error
}

func (m *Mysql) DeleteFuncInstanceByCondition(condition map[string]interface{}) error {
	return m.db.Unscoped().Where(condition).Delete(&model.FuncInstance{}).Error
}

func (m *Mysql) DeleteFuncInstance(id uint) error {
	return m.db.Unscoped().Delete(&model.FuncInstance{}, id).Error
}

func (m *Mysql) GetAwsTaskDefByFuncCpuMenImg(functionName string, cpu int32, memory int32, imageUrl string) (*model.AwsTaskDef, error) {
	data := &model.AwsTaskDef{}
	return data, m.db.Where(map[string]interface{}{
		"function_name": functionName,
		"cpu":           cpu,
		"memory":        memory,
		"image_url":     imageUrl,
	}).First(data).Error
}

func (m *Mysql) UpdateAwsTaskDef(data *model.AwsTaskDef) error {
	return m.db.Save(data).Error
}

func (m *Mysql) GetReqScheduleInfoByFunctionName(functionName string) ([]model.ReqScheduleInfo, error) {
	var data []model.ReqScheduleInfo
	return data, m.db.Where(map[string]interface{}{"function_name": functionName}).Find(&data).Error
}

func (m *Mysql) GetReqScheduleInfoByAwsServiceName(AwsServiceName string) ([]model.ReqScheduleInfo, error) {
	var data []model.ReqScheduleInfo
	return data, m.db.Where(map[string]interface{}{"placed_aws_service_name": AwsServiceName}).Find(&data).Error
}

func (m *Mysql) GetReqScheduleInfoByCondition(condition map[string]interface{}) ([]model.ReqScheduleInfo, error) {
	var data []model.ReqScheduleInfo
	return data, m.db.Where(condition).Find(&data).Error
}

func (m *Mysql) DeleteReqScheduleInfo(requestID uint64) error {
	return m.db.Unscoped().Where(map[string]interface{}{"req_id": requestID}).Delete(&model.ReqScheduleInfo{}).Error
}

func (m *Mysql) DeleteAllReqScheduleInfo() error {
	return m.db.Unscoped().Where("1 = 1").Delete(&model.ReqScheduleInfo{}).Error
}

func (m *Mysql) UpdateReqScheduleInfo(data *model.ReqScheduleInfo) error {
	return m.db.Save(data).Error
}
