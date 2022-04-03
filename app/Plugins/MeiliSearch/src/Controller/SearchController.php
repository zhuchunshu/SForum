<?php

namespace App\Plugins\MeiliSearch\src\Controller;

use App\CodeFec\Annotation\RouteRewrite;
use Hyperf\HttpServer\Annotation\Controller;
use Hyperf\HttpServer\Annotation\GetMapping;
use MeiliSearch\Client;

#[Controller]
class SearchController
{
	#[RouteRewrite(route:"/search")]
	public function index(){
		if(!request()->input('q')){
			return redirect()->url('/')->go();
		}
		// 客户端连接
		$client = new Client(get_options("meilisearch_url",'http://127.0.0.1:7700'),get_options("meilisearch_apikey",null));
		// 获取索引名
		$index = get_options("meilisearch_index",get_options("APP_NAME","SuperForum"));
		// 当前页码
		$page = request()->input('page',1);
		// offset操作
		if($page>1){
			$offset = ($page-1)*20;
		}else{
			$offset =0;
		}
		// 每页查询数量
		$limit = 15;
		// 搜索结果
		$data = $client->index($index)->search(request()->input('q'), [
			'attributesToHighlight' => ['title'],
			'offset' => $offset,
			'limit' => $limit
		])->getRaw();
		// 分页数量
		$page_num = ceil($data['nbHits']/20);
		return view("MeiliSearch::index",['data' => $data,'page_num' => $page_num]);
	}
	
}