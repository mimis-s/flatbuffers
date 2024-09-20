
# flatbuffers

* 类似于protobuffer的序列化工具, 序列化和反序列化速度比protobuffer快, protobuffer的数据结构设计比proto复杂, 使用也不方便, 不能直接把[]byte转换成想要的数据结构, 需要先转换成flatbuffers的内部结构, 再由这个结构转成我们想要的结构, 也没有protoc这种方便的插件功能添加我们想要的数据进去, 所以对于游戏外围业务我会考虑使用protobuffer, 生态好太多

* 但是由于公司项目中使用到了flatbuffer, 所以针对源码作了以下修改, 使它更符合我们的编码习惯
```
新增一个build.sh文件, 可以直接编译flatc, 并添加到环境变量中
flatbuffer源码, 修改了关于golang, go-grpc部分, 新增一个go-grpc文件
```

## flatbuffer-golang结构改动
*将之前需要table <-> 后缀"T" <-> []byte 这三种结构互相转换, 替换成了外部只需要后缀"T"结构, 加上pkg/flac/codec中的序列化函数
```
例:
在fbs文件中定义如下结构
table HelloRequest {
  name:string;
  list_id:[int32];
}
flatc自动生成以下两种结构:
type Request struct {
	_tab flatbuffers.Table
}
type RequestT struct {
	MsgId uint16 `json:"msg_id"`
	Data []byte `json:"data"`
}
之前的转换[]byte -> RequestT:
1: 先通过公共函数GetRootAs...将[]byte转换为Request结构
2: 调用Request的UnPack函数, 转换为RequestT结构
改变之后的转换(pkg/flac/codec中的序列化函数):
1: 在源码中增加RequestT的成员函数MarshalTable(flatc编译时自动完成)
2: codec.Unmarshal()将[]byte转换为RequestT

```
## flatbuffer-grpc结构改动
* 源码中的grpc调用复杂, 外部调用需要写很多序列化和反序列化代码:
```
定义一个grcp调用, 则生成如下代码:
OnMessage(ctx context.Context, in *flatbuffers.Builder,opts ...grpc.CallOption) (*Response, error)
这样的函数调用迫使外部调用者需要将结构转化为flatbuffers.Builder, 而且返回给用户的还是Response这种结构, 复杂且不实用
```
对于grcp做出如下修改:
```
OnMessage(ctx context.Context, in *RequestT,opts ...grpc.CallOption) (*ResponseT, error)
将输入输出都统一改成了我们熟悉的golang结构体, 外部调用简单直接, 不需要之前复杂冗余的解析了
```
* 新增一个grcp文件， 主要是方便外部调用做的封装, 尽可能让这些操作变成自动化

# lib

1. lib/flatc/codec是flatbuffer对应的序列化/反序列化工具, 让flatbuffer的序列化和proto一样方便

2. lib/flatc/local_grpc_stream是flatbuffer流stream的本地调用库(all_in_one环境)

3. lib/flatc/pools是grpc的一个客户端池子, 主要连接grpc服务器(这里的服务发现是直接连接service的域名, 通过域名解析达到和注册中心一样的效果), 所以flat+grpc这一套不需要考虑服务ip的租期续约等操作
