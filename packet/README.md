说明
====

这个包是link的分包型协议实现，和stream包不同点在于使用者只需要实现消息内容的封包解包，进一步可以通过实现binary.Spliter接口来做到自定义的分包方式。