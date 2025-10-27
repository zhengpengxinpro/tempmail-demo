package main

import (
	"database/sql"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
)

func main() {
	// 解析命令行参数
	dbType := flag.String("type", "", "数据库类型: mysql 或 postgres")
	dbDSN := flag.String("dsn", "", "数据库连接字符串")
	action := flag.String("action", "up", "操作: up (升级) 或 down (回滚)")
	flag.Parse()

	// 验证参数
	if *dbType == "" || *dbDSN == "" {
		fmt.Println("用法:")
		fmt.Println("  go run cmd/migrate/main.go -type=mysql -dsn='user:pass@tcp(host:port)/dbname' -action=up")
		fmt.Println("  go run cmd/migrate/main.go -type=postgres -dsn='postgres://user:pass@host:port/dbname' -action=up")
		os.Exit(1)
	}

	if *dbType != "mysql" && *dbType != "postgres" {
		fmt.Printf("错误: 不支持的数据库类型 '%s'\n", *dbType)
		os.Exit(1)
	}

	// 连接数据库
	db, err := sql.Open(*dbType, *dbDSN)
	if err != nil {
		fmt.Printf("错误: 无法连接数据库: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	// 测试连接
	if err := db.Ping(); err != nil {
		fmt.Printf("错误: 数据库连接失败: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✓ 成功连接到 %s 数据库\n", *dbType)

	// 读取迁移文件
	migrationFile := fmt.Sprintf("migrations/%s/001_initial_schema.%s.sql", *dbType, *action)
	
	// 获取当前工作目录
	wd, err := os.Getwd()
	if err != nil {
		fmt.Printf("错误: 无法获取工作目录: %v\n", err)
		os.Exit(1)
	}

	// 尝试多个可能的路径
	possiblePaths := []string{
		migrationFile,
		filepath.Join(wd, migrationFile),
		filepath.Join(wd, "..", "..", migrationFile),
	}

	var sqlContent []byte
	var foundPath string
	for _, path := range possiblePaths {
		content, err := os.ReadFile(path)
		if err == nil {
			sqlContent = content
			foundPath = path
			break
		}
	}

	if sqlContent == nil {
		fmt.Printf("错误: 找不到迁移文件\n")
		fmt.Printf("查找路径:\n")
		for _, path := range possiblePaths {
			fmt.Printf("  - %s\n", path)
		}
		os.Exit(1)
	}

	fmt.Printf("✓ 读取迁移文件: %s\n", foundPath)

	// 执行迁移
	fmt.Printf("执行 %s 操作...\n\n", *action)
	
	// 分割SQL语句
	stmts := splitStatements(string(sqlContent))
	fmt.Printf("找到 %d 条SQL语句\n\n", len(stmts))
	
	// 逐个执行SQL语句
	for i, stmt := range stmts {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" || strings.HasPrefix(stmt, "--") {
			continue
		}
		
		// 获取SQL首行用于显示
		firstLine := strings.Split(stmt, "\n")[0]
		if len(firstLine) > 60 {
			firstLine = firstLine[:60] + "..."
		}
		fmt.Printf("[%d/%d] %s\n", i+1, len(stmts), firstLine)
		
		if _, err := db.Exec(stmt); err != nil {
			fmt.Printf("\n错误: 执行迁移失败: %v\n", err)
			fmt.Printf("SQL: %s\n", stmt)
			os.Exit(1)
		}
	}

	fmt.Printf("\n✓ 迁移成功完成!\n")
}

// splitStatements 分割SQL语句（按分号分割，忽略字符串中的分号）
func splitStatements(sql string) []string {
	var statements []string
	var current strings.Builder
	var inString bool
	var stringChar rune

	for i, r := range sql {
		// 检查是否进入或退出字符串
		if r == '\'' || r == '"' || r == '`' {
			if !inString {
				inString = true
				stringChar = r
			} else if r == stringChar {
				inString = false
			}
			current.WriteRune(r)
		} else if r == ';' {
			current.WriteRune(r)
			if !inString {
				stmt := strings.TrimSpace(current.String())
				if stmt != "" && !strings.HasPrefix(stmt, "--") {
					statements = append(statements, stmt)
				}
				current.Reset()
			}
		} else {
			current.WriteRune(r)
		}

		// 如果是最后一个字符且buffer不为空
		if i == len(sql)-1 {
			stmt := strings.TrimSpace(current.String())
			if stmt != "" && !strings.HasPrefix(stmt, "--") {
				statements = append(statements, stmt)
			}
		}
	}

	return statements
}
