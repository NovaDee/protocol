# protocol

    //定义config文件进行配置
	log := &logger.Config{
		JSON:            false,
		Level:           "debug",
		ComponentLevels: map[string]string{},
	}

	logger.InitLogConfig(log)