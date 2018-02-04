# fproxy
高匿代理构建系统v0.1.0版本已完成：
1. 爬取指定代理网站数据，提取高匿ip及生成ip段用于扫描
2. 按指定的端口列表扫描ip段
3. 高匿检测，需提供公网http接口支持，目前使用nginx实现，在nginx中加入以下配置：
location /chkproxy.json {
		default_type 'application/json';
		content_by_lua_block {
			local via = ngx.header.via;
			local xfor = ngx.header.x_forwarded_for;
			if(via == nil and xfor == nil) then
				ngx.say('anony');
			else
				ngx.say('trans');
			end;
		}
	}

  
