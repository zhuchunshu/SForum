<?php

namespace App\Plugins\MeiliSearch\src\Controller;

use App\Middleware\AdminMiddleware;
use App\Model\AdminOption;
use App\Plugins\MeiliSearch\src\Data;
use Hyperf\HttpServer\Annotation\Controller;
use Hyperf\HttpServer\Annotation\GetMapping;
use Hyperf\HttpServer\Annotation\Middleware;
use Hyperf\HttpServer\Annotation\PostMapping;
use MeiliSearch\Client;

#[Controller(prefix:"/admin/MeiliSearch")]
#[Middleware(AdminMiddleware::class)]
class AdminController
{
	#[GetMapping(path:"")]
	public function index(){
		try {
			$client = new Client(get_options("meilisearch_url",'http://127.0.0.1:7700'),get_options("meilisearch_apikey",null));
			$status = $client->health()['status'];
			$code = 200;
		}catch(\Exception $e){
			$status = '连接错误';
			$code = 201;
		}
		return view("MeiliSearch::admin.index",['status'=>$status,'code'=>$code]);
	}
	
	#[PostMapping(path:"")]
	public function submit(){
		$url = request()->input('url');
		$apikey = request()->input('apikey');
		$index = request()->input('index');
		if(!$url && $index){
			return redirect()->url('/admin/MeiliSearch')->with('danger','请求参数不足!')->go();
		}
		$this->setOption([
			'meilisearch_url' => $url,
			'meilisearch_apikey' => $apikey,
			'meilisearch_index' => $index
		]);
		return redirect()->url('/admin/MeiliSearch')->with('success','修改成功!')->go();
	}
	
	private function setOption($data = []): void
	{
		foreach($data as $key =>$value){
			if(AdminOption::query()->where("name",$key)->exists()){
				AdminOption::query()->where("name",$key)->update(['value'=>$value]);
			}else{
				AdminOption::query()->create(['name' => $key,'value'=>$value]);
			}
		}
		options_clear();
	}
	
	#[GetMapping(path:"push")]
	public function test(){
		$data = new Data();
		$client = new Client(get_options("meilisearch_url",'http://127.0.0.1:7700'),get_options("meilisearch_apikey",null));
		$index = get_options("meilisearch_index",get_options("APP_NAME","SuperForum"));
		$client->deleteIndex($index);
		$client->index($index)->addDocuments($data->get());
		$client->index($index)->updateSearchableAttributes([
			'title',
		]);
	}
	
}