#铁屑

自己用的分表功能包

###使用说明
在表需要初始化的地方：
~~~
    db, e := gorm.Open(
		config.DB.DriverName,
		config.DB.User+`@`+
			`(`+config.DB.Host+`)`+
			`/`+config.DB.Database+`?charset=utf8`)

	db.DB().SetMaxIdleConns(4)
	db.DB().SetMaxOpenConns(20)
	db.DB().SetConnMaxLifetime(8 * time.Second)
	db.LogMode(config.DB.Debug)
ironShard := IronShard.NewShard(db, config.DB.Database)
tbsql := "  `id` bigint NOT NULL AUTO_INCREMENT," +
         "  `open_id` varchar(100) COLLATE utf8mb4_general_ci DEFAULT NULL, " +
         "  `head_url` varchar(500) COLLATE utf8mb4_general_ci DEFAULT NULL," +
         "  `country` varchar(100) COLLATE utf8mb4_general_ci DEFAULT NULL," +
         "  `create_ip` varchar(15) COLLATE utf8mb4_general_ci DEFAULT NULL," +
         "  `create_ip_v6` varchar(255) COLLATE utf8mb4_general_ci DEFAULT NULL," +
         "  `nick_name` varchar(200) COLLATE utf8mb4_general_ci DEFAULT NULL," +
         "  `channel` int DEFAULT NULL,  `platform` varchar(255) COLLATE utf8mb4_general_ci DEFAULT NULL," +
         "  `forbidden` int DEFAULT NULL COMMENT '默认0正常  1-帐号被禁 2-时限禁用'," +
         "  `forbidden_end` timestamp(3) NULL DEFAULT NULL COMMENT '如果forb为2到时间结束禁用结束'," +
         "  `role` int DEFAULT NULL COMMENT '角色0-玩家  1-测试人员',"
ironShard.InitAtStartServer("user_m", tbsql, false)
~~~



