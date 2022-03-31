# callByCSV

> 一个通过csv元数据来自动刷数据的工具\
解决产研每次导出原始数据，再写脚本通过解析原始数据来一行一行的刷数据开发慢问题

## 使用方法
1. 先导出原始csv格式文件
2. 编辑csv格式文件，在前10行加上配置项
```csv
url,http://192.168.7.171:8099/duxuesc/api/introtransupdate
perSecond,1
PerCount,100
batch,0
cookie,ZYBIPSCAS=IPS_****
```
3. 修改csv文件编码格式为`utf-8无BOM`格式。
4. 使用本执行程序,传入参数元数据文件路径 
```shell
callByCsv.exe  data.csv 
```

### 注意：
1. csv文件必须是utf-8无bom格式。否则拼装的URL错误。
2. csv的表头字段是最终形成url的参数中的key，需注意不能写错。
