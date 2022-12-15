<?php

namespace App\Controller\Admin;

use App\Jobs\Upgrading;
use App\Middleware\AdminMiddleware;
use App\Plugins\Topic\src\ContentParse;
use Hyperf\Di\Annotation\Inject;
use Hyperf\HttpServer\Annotation\Controller;
use Hyperf\HttpServer\Annotation\Middleware;
use Hyperf\HttpServer\Annotation\PostMapping;
use Hyperf\Utils\Str;
use function Swoole\Coroutine\Http\get;

#[Controller(prefix:"/api/admin")]
#[Middleware(AdminMiddleware::class)]
class ApiController
{
	private string $api_releases = "https://api.github.com/repos/zhuchunshu/super-forum/releases";
	private string $update_log = "https://www.runpod.cn/48.md";
	#[PostMapping(path:"getVersion")]
	public function getVersion(){
		// 获取最新版
		if(!cache()->has('admin.git.getVersion')){
			$data = http()->get($this->api_releases);
			$data = $data[0];
			
			// 获取当前程序版本信息
			$build_info = include BASE_PATH."/build-info.php";
			$data = array_merge($data, $build_info);
			$version = $data['version'];
			$tag_name = $data['tag_name'];
			
			// 判断是否可升级
			$data['upgrade']=false;
			if($tag_name >$version && $data['prerelease']===false){
				$data['upgrade']=true;
			}
			
			// 其他
			$data['new_version_url'] = "/admin/Releases/".$data['id'];
			
			// 返回数据
			cache()->set('admin.git.getVersion',$data,600);
		}
		return cache()->get('admin.git.getVersion');
		
	}
	#[PostMapping(path:"getRelease/{id}")]
	public function getRelease($id){
		if(!cache()->has('admin.git.getRelease.'.$id)){
			cache()->set('admin.git.getRelease.'.$id,http()->get($this->api_releases."/".$id),600);
		}
		return cache()->get('admin.git.getRelease.'.$id);
	}
	
	#[PostMapping(path:"getUpdateLog")]
	public function getCommit()
    {
        $data = http('raw')->get($this->update_log)->getBody()->getContents();
		return (new ContentParse())->parse($data);
	}
	
	#[PostMapping(path:"clearCache")]
	public function clearCache(){
		cache()->delete('admin.git.getVersion');
		return Json_Api(200,true,['msg' => '缓存清理成功']);
	}
	
	
	/**
	 * @Inject
	 * @var Upgrading
	 */
	protected $service;
	
	#[PostMapping(path:"update")]
	public function update(){
		$url = match((string)get_options('update_server',2)){
			'2' => '',
			'1' => 'https://ghproxy.com/'
		};
		$data = http()->get($this->api_releases);
		$data = $data[0];
		
		// 获取当前程序版本信息
		$build_info = include BASE_PATH."/build-info.php";
		$data = array_merge($data, $build_info);
		$version = $data['version'];
		$tag_name = $data['tag_name'];
		
		// 判断是否不可升级
		if($tag_name <=$version || $data['prerelease']===true){
			return Json_Api(403,false,['msg' => '无需升级!']);
		}
		
		// 生成文件下载链接
		$url .= "https://github.com/zhuchunshu/super-forum/archive/".$tag_name.".zip";
		
		// 定义文件存放路径
		$file_path= BASE_PATH."/runtime/update.zip";
		
		// 创建下载任务
		$this->service->handle($url,$file_path);
		return Json_Api(200,true,['msg' => '升级任务已创建']);
	}
}