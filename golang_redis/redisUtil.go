package main

import (
	"encoding/json"
	"errors"
	"github.com/garyburd/redigo/redis"
	"github.com/golang/protobuf/proto"
	"log"
	"reflect"
	"sync"
	"time"
)

var (
	m      sync.RWMutex //互斥锁
	status bool
	pool   *redis.Pool
	once   sync.Once
)

// RedisConfig Redis 配置
type RedisConfig struct {
	RedisUrl            string `json:"redisUrl,omitempty" yaml:"redisUrl"`           //Redis 地址
	RedisPassword       string `json:"redisPassword,omitempty" yaml:"redisPassword"` //Redis Login 密码
	RedisDB             int    `json:"redisDB,omitempty" yaml:"redisDB"`             //Redis DB
	RedisTimeout        int    `json:"redisTimeout,omitempty" yaml:"redisTimeout"`   //Redis 超时时间
	RedisMaxActive      int    `json:"redisMaxActive,omitempty" yaml:"redisMaxActive"`
	RedisMaxIdle        int    ` json:"redisMaxIdle,omitempty"yaml:"maxIdle"`
	RedisMaxIdleSeconds int    `json:"redisMaxIdleSeconds,omitempty" yaml:"redisMaxIdleSeconds"`
	RedisWaitExhaust    bool   `json:"redisWaitExhaust,omitempty" yaml:"redisWaitExhaust"`
}

// InitRedis Redis Init
func InitRedis(redisConfig *RedisConfig) {
	m.Lock()
	defer m.Unlock()

	//1.如果redis 初始化了 则 return
	if status {
		log.Println("[InitRedis] Redis已初始化")
	}
	log.Println("[InitRedis] Redis初始化中")

	pool = newPool(redisConfig)
	//Set status
	status = true
	log.Println("[InitRedis] Redis初始化 成功", redisConfig.RedisUrl)
}

func newPool(cfg *RedisConfig) *redis.Pool {
	server := cfg.RedisUrl
	password := cfg.RedisPassword
	timeout := time.Duration(cfg.RedisTimeout) * time.Second //10
	maxActive := cfg.RedisMaxActive                          // 1000
	maxIdle := cfg.RedisMaxIdle                              // 1000
	maxIdleSeconds := cfg.RedisMaxIdleSeconds                //3600
	waitExhaust := cfg.RedisWaitExhaust                      // false
	redisdb := cfg.RedisDB
	return &redis.Pool{
		MaxIdle:     maxIdle,
		MaxActive:   maxActive,
		IdleTimeout: time.Duration(maxIdleSeconds) * time.Second,
		Wait:        waitExhaust,
		Dial: func() (redis.Conn, error) {
			c, err := redis.DialTimeout("tcp", server, timeout, timeout, timeout)
			if err != nil {
				panic(err.Error())
				return nil, err
			}
			if len(password) > 0 {
				_, err = c.Do("AUTH", password)
				if err != nil {
					return nil, err
				}
			}
			if redisdb > 0 {
				_, err = c.Do("SELECT", redisdb)
				if err != nil {
					return nil, err
				}
			}
			return c, err
		},
		TestOnBorrow: func(c redis.Conn, t time.Time) error {
			_, err := c.Do("PING")
			return err
		},
	}
}

func GetPool() *redis.Pool {
	return getPool()
}

func getPool() *redis.Pool {
	if pool == nil {
		log.Printf("Redis Pool is empty, trying initialize it")
	}

	return pool
}

var SetObject = func(key string, value interface{}) error {
	conn := getPool().Get()
	defer conn.Close()

	if _, err := do(conn, "HMSET", redis.Args{}.Add(key).AddFlat(value)...); err != nil {
		log.Printf("redisUtil SetObject error:", err)
		return err
	}
	return nil
}

var GetObject = func(key string, value interface{}) (err error) {
	conn := getPool().Get()
	defer conn.Close()
	v, err := redis.Values(do(conn, "HGETALL", key))
	if err != nil {
		log.Printf("redisUtil GetObject error:%v", err)
		return err
	}

	//log.Println("redisUtil GetObject v:", v)

	if err := redis.ScanStruct(v, value); err != nil {
		log.Printf("Redist Util Error for getting key [%s], error is [%+v]", key, err)
		return err
	}
	//log.Println("redisUtil GetObject value:", value)
	return nil
}

//设置key多少秒后超时
func Expire(key string, seconds int) error {
	conn := getPool().Get()
	defer conn.Close()
	_, err := do(conn, "EXPIRE", key, seconds)
	if err != nil {
		log.Printf("redisUtil Expire error: %v", err)
	}
	return nil
}

func Delete(key string) error {
	conn := getPool().Get()
	defer conn.Close()
	if _, err := do(conn, "DEL", key); err != nil {
		log.Printf("redisUtil Delete error:%v", err)
		return err
	}
	return nil
}

func SetObjectWithExpire(key string, value interface{}, expire int) error {
	conn := getPool().Get()
	defer conn.Close()
	if _, err := do(conn, "HMSET", redis.Args{}.Add(key).AddFlat(value)...); err != nil {
		log.Printf("redisUtil SetObject error:%v", err)
		return err
	}
	Expire(key, expire)
	return nil
}

func SetStringWithExpire(key string, value string, expire int) error {
	conn := getPool().Get()
	defer conn.Close()
	if _, err := do(conn, "SET", key, value, "EX", expire); err != nil {
		log.Printf("redisUtil SetStringWithExpire error:%v", err)
		return err
	}
	return nil
}

func SetString(key string, value string) error {
	conn := getPool().Get()
	defer conn.Close()
	if _, err := do(conn, "SET", key, value); err != nil {
		log.Printf("redisUtil SetStringWithExpire error:%v", err)
		return err
	}
	return nil
}

func GetString(key string) (string, error) {
	conn := getPool().Get()
	defer conn.Close()
	v, err := redis.String(do(conn, "GET", key))
	if err != nil {
		if err == redis.ErrNil {
			return "", nil
		}
		return "", err
	}
	return v, nil
}

func Exists(key string) bool {
	conn := getPool().Get()
	defer conn.Close()
	v, err := redis.Bool(do(conn, "EXISTS", key))
	if err != nil {
		log.Printf("redisUtil Exist error:%v", err)
		return false
	}
	return v
}

func AddGeoIndex(indexName string, geoKey string, latitude float32, longitude float32) error {
	conn := getPool().Get()
	defer conn.Close()
	_, err := do(conn, "GEOADD", indexName, latitude, longitude, geoKey)
	if err != nil {
		log.Printf("add geo index error:%v", err)
		return err
	}
	return nil
}

func GetKeysByPrefix(prefix string) ([]string, error) {
	conn := getPool().Get()
	defer conn.Close()
	keys, err := redis.Strings(do(conn, "KEYS", prefix+"*"))
	if err != nil {
		return nil, err
	}
	return keys, nil
}

//往redis里面插入键值，如果键已存在，返回False，不执行
//如果键不存在，插入键值,返回成功
func SetStringIfNotExist(key, value string, expire int) (bool, error) {
	conn := getPool().Get()
	defer conn.Close()
	result, err := redis.String(do(conn, "SET", key, value, "EX", expire, "NX"))
	if err != nil {
		if err == redis.ErrNil {
			return false, nil
		} else {
			log.Printf("redisUtil SetStringWithExpire error:%v", err)
			return false, err
		}
	}
	if result == "OK" {
		return true, nil
	} else {
		return false, nil
	}
}

var (
	errKeyIsBlank        = errors.New("redis: key should not be blank")
	errValueIsNotPointer = errors.New("redis: value should be non-nil pointer")
	errValueIsNil        = errors.New("redis: value should not be nil")
	ErrKeyNotFound       = errors.New("redis: key not found or expired")
)

// GetValue key should not be blank and value must be non-nil pointer to int/bool/string/struct ...
// because a value can set only if it is addressable
// if key not found will return ErrKeyNotFound
func GetValue(key string, value interface{}) (err error) {
	if len(key) == 0 {
		return errKeyIsBlank
	}
	v := reflect.ValueOf(value)
	if v.Kind() != reflect.Ptr {
		return errValueIsNotPointer
	}
	if v.IsNil() {
		return errValueIsNil
	}

	conn := getPool().Get()
	defer conn.Close()

	reply, err := redis.Values(do(conn, "MGET", key))
	if err != nil {
		log.Printf("redis: get %s %v", key, err)
		return err
	}
	if len(reply) == 0 || reply[0] == nil {
		return ErrKeyNotFound
	}

	if v.Elem().Kind() == reflect.Struct {
		err = json.Unmarshal(reply[0].([]byte), value)
	} else {
		_, err = redis.Scan(reply, value)
	}
	if err != nil {
		log.Printf("redis: scan %s %v", key, err)
		return err
	}
	log.Println("redis: get %s success", key)
	return nil
}

// SetValue key should not be blank and value should not be nil
// Struct or pointer to struct values will encoding as JSON objects
func SetValue(key string, value interface{}, seconds ...int) (err error) {
	if len(key) == 0 {
		return errKeyIsBlank
	}
	v := reflect.ValueOf(value)
	isPtr := (v.Kind() == reflect.Ptr)
	if value == nil || (isPtr && v.IsNil()) {
		return errValueIsNil
	}

	conn := getPool().Get()
	defer conn.Close()

	if v.Kind() == reflect.Struct || (isPtr && v.Elem().Kind() == reflect.Struct) {
		bs, err := json.Marshal(value)
		if err != nil {
			return err
		}
		value = string(bs)
	} else {
		if isPtr { // *int/*bool/*string ...
			value = v.Elem()
		}
	}
	exArgs := []interface{}{}
	if len(seconds) > 0 {
		exArgs = append(exArgs, "EX", seconds[0])
	}
	args := append([]interface{}{key, value}, exArgs...)
	_, err = do(conn, "SET", args...)
	if err != nil {
		log.Printf("redis: set %s ", key, err)
		return err
	}
	log.Println("redis: set %s success", key)
	return nil
}

func do(c redis.Conn, commandName string, args ...interface{}) (reply interface{}, err error) {
	reply, err = c.Do(commandName, args...)
	if err != nil {
		handleAlertError(err)
	}
	return reply, err
}

func handleAlertError(err error) {
	if err == nil {
		return
	}
	log.Println("send slack alert")
}

func SetStrings(key string, ss []string, seconds ...int) error {
	if len(key) == 0 {
		return errKeyIsBlank
	}

	conn := getPool().Get()
	defer conn.Close()
	var err error
	for _, s := range ss {
		if err = conn.Send("SADD", key, s); err != nil {
			log.Printf("redis: Send %v", err)
			return err
		}
	}
	if err = conn.Flush(); err != nil {
		log.Printf("redis: Flush %v", err)
		return err
	}
	_, err = conn.Do("")
	if err != nil {
		log.Printf("redis: SADD %s %v", key, ss, err)
		return err
	}
	if len(seconds) > 0 {
		_, err = conn.Do("EXPIRE", key, seconds[0])
		if err != nil {
			log.Printf("redis: EXPIRE %s %v", key, seconds[0], err)
			return err
		}
	}
	log.Println("redis: SetStrings %s success", key)
	return nil
}

func GetStrings(key string) ([]string, error) {
	if len(key) == 0 {
		return nil, errKeyIsBlank
	}

	conn := getPool().Get()
	defer conn.Close()
	ss, err := redis.Strings(conn.Do("SMEMBERS", key))
	if err != nil {
		log.Printf("redis: SMEMBERS %s ", key, err)
		return nil, err
	}
	log.Println("redis: GetStrings %s success", key)
	return ss, nil
}

func Incr(key string) (*int, error) {
	if len(key) == 0 {
		return nil, errKeyIsBlank
	}

	conn := getPool().Get()
	defer conn.Close()

	id, err := redis.Int(conn.Do("INCR", key))
	if err != nil {
		log.Printf("redisUtil INCR error:", err)
		return nil, err
	}
	log.Println("redis: Incr %s success, %v", key, id)
	return &id, nil
}

//SET if Not exists
func SetValueNX(key string, value interface{}, seconds ...int) (err error) {
	if Exists(key) {
		return nil
	}
	if len(key) == 0 {
		return errKeyIsBlank
	}
	v := reflect.ValueOf(value)
	isPtr := (v.Kind() == reflect.Ptr)
	if value == nil || (isPtr && v.IsNil()) {
		return errValueIsNil
	}

	conn := getPool().Get()
	defer conn.Close()

	if v.Kind() == reflect.Struct || (isPtr && v.Elem().Kind() == reflect.Struct) {
		bs, err := json.Marshal(value)
		if err != nil {
			return err
		}
		value = string(bs)
	} else {
		if isPtr { // *int/*bool/*string ...
			value = v.Elem()
		}
	}
	exArgs := []interface{}{}
	if len(seconds) > 0 {
		exArgs = append(exArgs, "EX", seconds[0])
	}
	args := append([]interface{}{key, value}, exArgs...)
	_, err = do(conn, "SET", args...)
	if err != nil {
		log.Printf("redis: set %s ", key, err)
		return err
	}
	log.Println("redis: set %s success", key)
	return nil
}

func HIncrBy(key, field string, value int) (int, error) {
	if len(key) == 0 {
		return -1, errKeyIsBlank
	}

	conn := getPool().Get()
	defer conn.Close()

	result, err := redis.Int(conn.Do("HINCRBY", key, field, value))
	if err != nil {
		log.Printf("redisUtil HINCRBY error:%v", err)
		return -1, err
	}
	log.Println("redis: HINCRBY %s success, %v", key, result)
	return result, nil
}

// PopValue 从队列头部取值
func PopValue(key string, value interface{}, isJSON bool) (err error) {
	if len(key) == 0 {
		return errKeyIsBlank
	}
	v := reflect.ValueOf(value)
	if v.Kind() != reflect.Ptr {
		return errValueIsNotPointer
	}
	if v.IsNil() {
		return errValueIsNil
	}

	conn := getPool().Get()
	defer conn.Close()

	reply, err := redis.Bytes(do(conn, "LPOP", key))
	if err != nil {
		//log.Printf("redis: LPOP %s %v", key, err)
		return err
	}

	if v.Elem().Kind() == reflect.Struct {
		if isJSON {
			err = json.Unmarshal(reply, value)
		} else {
			err = proto.Unmarshal(reply, value.(proto.Message))
		}
	} else {
		_, err = redis.Scan(append([]interface{}{}, reply), value)
	}
	if err != nil {
		log.Printf("redis: LPOP %s %v", key, err)
		return err
	}
	log.Println("redis: LPOP %s success", key)
	return nil
}

// PushValue 从队列尾部插入值
func PushValue(key string, value interface{}, isJSON bool) (err error) {
	if len(key) == 0 {
		return errKeyIsBlank
	}
	v := reflect.ValueOf(value)
	isPtr := (v.Kind() == reflect.Ptr)
	if value == nil || (isPtr && v.IsNil()) {
		return errValueIsNil
	}

	conn := getPool().Get()
	defer conn.Close()

	if v.Kind() == reflect.Struct || (isPtr && v.Elem().Kind() == reflect.Struct) {
		var bs []byte
		var err1 error
		if isJSON {
			bs, err1 = json.Marshal(value)
		} else {
			bs, err1 = proto.Marshal(value.(proto.Message))
		}
		if err1 != nil {
			return err1
		}
		value = string(bs)
	} else {
		if isPtr { // *int/*bool/*string ...
			value = v.Elem()
		}
	}
	args := []interface{}{key, value}
	_, err = do(conn, "RPUSH", args...)
	if err != nil {
		log.Println("redis: RPUSH %s ", key, err)
		return err
	}
	log.Println("redis: RPUSH %s success", key)
	return nil
}
