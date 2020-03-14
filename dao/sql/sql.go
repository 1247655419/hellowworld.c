// Package sql ...
// sql业务语句
package sql

/**************online used**********************************************/

// GsqlGetDnGroupInfo 域名信息查询语句
const GsqlGetDnGroupInfo = `SELECT 
domain, 
province, 
popid, 
nodeid, 
group_id, 
node_type, 
upper_sub_node_id,
is_master
FROM T_OMS_VIEW_DOMAIN_NODE_GROUP`

// GsqlGetTopoInfo topo信息查询语句
const GsqlGetTopoInfo = `SELECT 
report_node_id, 
node_name, 
report_node_type, 
sub_node_id, 
upper_sub_node_id,
group_id, 
service_vip_addr, 
service_vipv6_addr 
FROM T_OMS_VIEW_NODE_INFO`

// GsqlGetCPInfo 查询cpid列表
const GsqlGetCPInfo = `SELECT CPID FROM T_OMS_VIEW_CPINFO`

// GsqlGetMessage1099 查询1099报文内容
const GsqlGetMessage1099 = `SELECT 
cpid,
cpname,
name,
url,
cache_switch,
cache_ttl,
auth_id,
cache_index,
cache_update,
share_server_connection,
download_url,
backup_download_url,
pull_parameter_switch,
offset_type,
offset_start,
offset_end,
complete_download,
complete_download_threshold,
request_headers
FROM T_OMS_VIEW_CP_CACHEPOLICY`
/**********************************
cache_ttl,
auth_id,
'cache_index',
'cache_update',
'share_server_connection',
'download_url',
'backup_download_url',
'pull_parameter_switch',
'offset_type',
'offset_start',
'offset_end',
'complete_download',
'complete_download_threshold',
'authid',
'request_headers'
FROM T_OMS_VIEW_CP_CACHEPOLICY`
**********************************/

// GsqlGetMessage10046 查询10046报文内容
const GsqlGetMessage10046 = `SELECT 
spid,
spname,
spservicestate
FROM T_OMS_VIEW_CP_DETAIL_INFO
`

// GsqlGetMessage10047 查询10047报文内容
const GsqlGetMessage10047 = `SELECT 
ownerspid,
deliverydnid,
deliverydn,
servicestate,
businesstype,
preferipmode,
sourcespriority,
protocol,
sources
FROM T_OMS_VIEW_DOMAIN
` 

/***************online used**********************************************/

/**************DEBUG******************
const GsqlGetDnGroupInfo = `SELECT 
domain, 
province, 
popid, 
nodeid, 
group_id, 
node_type, 
upper_sub_node_id,  
is_master
FROM domain_clouddragon`

const GsqlGetTopoInfo = `SELECT 
report_node_id, 
node_name, 
report_node_type, 
sub_node_id, 
upper_sub_node_id,
group_id, 
service_vip_addr, 
service_vipv6_addr 
FROM deploy_clouddragon`

const GsqlGetCPInfo = `SELECT CPID FROM T_OMS_VIEW_CPINFO`
**************DEBUG******************/
