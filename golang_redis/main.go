package main

import (
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"log"
)

func main() {
	//1.读取配置文件
	file, err := ioutil.ReadFile("server.yaml")
	if err != nil {
		log.Println(err)
		panic(err)
	}
	var redisConfig RedisConfig
	err = yaml.Unmarshal(file, &redisConfig)
	if err != nil {
		log.Println(err)
		panic(err)
	}

	InitRedis(&redisConfig)

	SetString("apples", "你哈")
	getString, err := GetString("apples")
	if err != nil {
		return
	}
	log.Println(getString)

	SetObject("redis", &redisConfig)

	//var redis RedisConfig
	//GetObject("redis", &redis)
	//
	//log.Println(redis)

}
