<?php

namespace App\Controller\Admin;

use App\Middleware\AdminMiddleware;
use Hyperf\HttpServer\Annotation\Controller;
use Hyperf\HttpServer\Annotation\Middleware;
use Hyperf\HttpServer\Annotation\PostMapping;

#[Controller(prefix:"/api/admin")]
#[Middleware(AdminMiddleware::class)]
class ApiController
{
	private string $api_releases = "https://api.github.com/repos/zhuchunshu/super-forum/releases";
	private string $api_commit = "https://github-api.inkedus.workers.dev/repos/zhuchunshu/super-forum/commits?per_page=10000";
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
			if($tag_name >$version){
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
	
	#[PostMapping(path:"getCommit")]
	public function getCommit(){
		if(!cache()->has('admin.git.commit')){
			cache()->set('admin.git.commit',http()->get($this->api_commit),600);
		}
		return cache()->get('admin.git.commit');
	}
}