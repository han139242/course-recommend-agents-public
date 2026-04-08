package main

import (
	"fmt"
	"log"

	"github.com/han139242/course-recommend-agents/backend/internal/config"
	"github.com/han139242/course-recommend-agents/backend/internal/module/auth/service"
	classroommodel "github.com/han139242/course-recommend-agents/backend/internal/module/teacher/model"
	usermodel "github.com/han139242/course-recommend-agents/backend/internal/module/user/model"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// main 是开发演示数据初始化入口。
// 该命令只面向本地或受控环境，用于快速准备最小可运行数据集。
func main() {
	cfg, err := config.LoadSeedConfig()
	if err != nil {
		log.Fatalf("加载 seed 配置失败: %v", err)
	}

	db, err := gorm.Open(mysql.Open(cfg.MySQL.DSN), &gorm.Config{})
	if err != nil {
		log.Fatalf("连接 MySQL 失败: %v", err)
	}

	passwordService := service.NewPasswordService()
	if err := seedDemoData(db, passwordService); err != nil {
		log.Fatalf("写入开发演示数据失败: %v", err)
	}

	fmt.Println("开发环境演示数据已初始化完成")
}

// seedDemoData 幂等写入演示账号与班级数据。
// 通过先查后建的方式重复执行命令时不会制造重复记录。
func seedDemoData(db *gorm.DB, passwordService *service.PasswordService) error {
	teacherEmail := "teacher@example.com"
	studentEmail := "student@example.com"
	classroomName := "2026 春人工智能 1 班"
	grade := "2026 春"
	studentNo := "20260001"
	title := "副教授"

	teacherPasswordHash, err := passwordService.Hash("123456")
	if err != nil {
		return err
	}
	studentPasswordHash, err := passwordService.Hash("123456")
	if err != nil {
		return err
	}

	teacher := usermodel.User{}
	if err := db.Where("email = ?", teacherEmail).First(&teacher).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			teacher = usermodel.User{
				Name:         "张老师",
				Email:        teacherEmail,
				PasswordHash: teacherPasswordHash,
				Role:         "teacher",
				Title:        &title,
				Target:       "",
				Status:       "active",
			}
			if err := db.Create(&teacher).Error; err != nil {
				return err
			}
		} else {
			return err
		}
	}

	classroom := classroommodel.Classroom{}
	if err := db.Where("name = ? AND teacher_id = ?", classroomName, teacher.ID).First(&classroom).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			classroom = classroommodel.Classroom{
				Name:      classroomName,
				Grade:     &grade,
				TeacherID: teacher.ID,
				Status:    "active",
			}
			if err := db.Create(&classroom).Error; err != nil {
				return err
			}
		} else {
			return err
		}
	}

	student := usermodel.User{}
	if err := db.Where("email = ?", studentEmail).First(&student).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			student = usermodel.User{
				Name:         "李同学",
				Email:        studentEmail,
				PasswordHash: studentPasswordHash,
				Role:         "student",
				StudentNo:    &studentNo,
				ClassID:      &classroom.ID,
				Target:       "计算机视觉",
				Status:       "active",
			}
			if err := db.Create(&student).Error; err != nil {
				return err
			}
		} else {
			return err
		}
	}

	return nil
}
