<?php

namespace App\Controller;

use App\Middleware\AdminMiddleware;
use App\Model\AdminOption;
use Hyperf\HttpServer\Annotation\{Controller, GetMapping, Middleware, PostMapping};

#[Controller(prefix:"/admin/themes")]
#[Middleware(AdminMiddleware::class)]
class ThemesController
{
	// 主题管理 - 首页
	#[GetMapping(path:"")]
	public function index(){
		return view("admin.themes.index");
	}
	
	// 主题信息
	#[PostMapping(path:"")]
	public function get(){
		return [
			'enable' => get_options("theme","CodeFec"),
		];
	}
	
	// 数据迁移
	#[PostMapping("Migrate")]
	public function Migrate($name=null): array
	{
		if(!$name){
			if (!request()->input("name")) {
				return Json_Api(403, false, ['msg' => '主题名不能为空']);
			}
			
			$theme_name = request()->input("name");
		}else{
			$theme_name = $name;
		}
		
		if (is_dir(theme_path($theme_name . "/resources/views")) && !is_dir(BASE_PATH . "/resources/views/themes")) {
			//return Json_Api(200,true,['msg' => BASE_PATH."/resources/views/plugins/".$plugin_name]);
			\Swoole\Coroutine\System::exec("mkdir " . BASE_PATH . "/resources/views/themes");
		}
		if (is_dir(theme_path($theme_name . "/resources/assets"))) {
			if (!is_dir(public_path("plugins"))) {
				mkdir(theme_path("plugins"));
			}
			if (!is_dir(public_path("themes/" . $theme_name))) {
				mkdir(public_path("themes/" . $theme_name));
			}
			copy_dir(theme_path($theme_name . "/resources/assets"), public_path("plugins/" . $theme_name));
		}
		return Json_Api(200, true, ['msg' => '资源迁移成功!']);
	}
	
	// 迁移所有资源
	#[PostMapping(path:"MigrateAll")]
	public function MigrateAll(): array
	{
		foreach (Plugins_EnList() as $name){
			$this->Migrate($name);
		}
		return Json_Api(200, true, ['msg' => '资源迁移成功!']);
	}
	
	// 卸载主题
	#[PostMapping(path:"remove")]
	public function remove(){
		$path = theme_path(request()->input('name'));
		if ($path && is_dir($path)) {
			\Swoole\Coroutine\System::exec("rm -rf " . $path);
			if(\Hyperf\Utils\Str::is('Linux',system_name())){
				\Swoole\Coroutine\System::exec("yes | composer du");
			}else{
				\Swoole\Coroutine\System::exec("composer du");
			}
			return Json_Api(200, true, ['msg' => "卸载成功!"]);
		}
		
		return Json_Api(403, false, ['msg' => "卸载失败,目录:" . $path . " 不存在!"]);
	}
	
	// 启用主题
	#[PostMapping(path:"enable")]
	public function enable(){
		$name =request()->input('name');
		if(theme()->has($name)){
			$this->setOption([
				'theme' => $name,
			]);
			return Json_Api(200, true, ['msg' => "主题启用成功!"]);
		}
		
		return Json_Api(403, false, ['msg' => "启用失败,主题:" . $name . " 不存在!"]);
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
	
}