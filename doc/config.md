# 配置 (Config)

框架提供了强大且可扩展的配置功能，支持以下功能:
1. **多环境配置** 根据环境配置加载不同的配置
2. **加载环境变量** 配置可通过环境变量传入
3. **配置加密** 配置可加密存放于配置文件中
4. **自定义配置** 框架提供了BaseConfig类型的配置类型, 项目可按需自行拓展
5. **指定配置路径** 配置可与代码分离, 通过路径的方式传入, 再读取具体路径的配置文件, 具体可见: [启动参数](./args.md)

## 多环境配置
框架支持根据环境来加载配置，定义多个环境的配置文件，具体可见: [环境](./env.md)

配置目录下, 可存放多个环境的配置, 框架会根据环境加载不同的配置。
```bash
config
├ config.default.yml # 默认配置
├ config.prod.yml    # prod环境下, 加载该配置
├ config.test.yml    # test环境下, 加载该配置
└ config.dev.yml     # dev环境下, 加载该配置
```

`config.default.yml`为默认配置, 所有环境都会先加载该配置文件, 然后再读取具体环境对应的配置, **相同配置项, 会覆盖默认配置的配置项**。

如: 当前环境为`prod`, 且有以下配置
```yml
# config.default.yml
service:
  port: 7777
````

```yml
# config.prod.yml
service:
  port: 7778
```

最终应用中的`service`的`port`为7778

## 加载环境变量
对于接入k8s等支持配置加密变量的项目, 可将一些不便直接存放在配置中的配置项配置到k8s的密钥中, 框架将通过读取环境变量的方式加载并替换配置, 
框架将会根据`{{xxx}}`中的变量名, 从环境变量中获取对应的值, 并替换到配置中.

如: 项目中的mysql配置信息如下：
|host|port|username|password|
|--|--|--|--|
|127.0.0.1|3306|root|password|

配置环境变量如下：
|dbHost|dbPort|dbUsername|dbPassword|
|--|--|--|--|
|127.0.0.1|3306|root|password|

```yml
db:
  host: "{{dbHost}}"
  port: {{dbPort}}
  username: "{{dbUsername}}"
  password: "{{dbPassword}}"
```


## 配置加密
考虑到不是所有项目都接入了k8s, 且环境变量配置稍显复杂, 故框架支持对配置中的参数进行加密存放。此时需要在运行项目时, 在命令行参数中加入`cipherKey`, 框架将会使用命令行中的cipherKey作为解密密钥, 对`CIPHER(xxx)`中的`xxx`进行解密, 并替换到配置中.
> 加密方式为aes

如：mysql的password为 `Hello World`, aes的密钥为 `UTabIUiHgDyh464+`
```yml
db:
  password: CIPHER(/t8wxJyz5nLKYDa7w8W3oQ==)
```
