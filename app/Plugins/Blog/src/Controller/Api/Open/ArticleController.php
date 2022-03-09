<?php

namespace App\Plugins\Blog\src\Controller\Api\Open;

use App\Plugins\Blog\src\Models\Blog;
use App\Plugins\Blog\src\Models\BlogArticle;
use App\Plugins\Blog\src\Models\BlogClass;
use Hyperf\HttpServer\Annotation\Controller;
use Hyperf\HttpServer\Annotation\PostMapping;

#[Controller(prefix:"/api/Blog/open/article")]
class ArticleController
{
	#[PostMapping(path:"data")]
	public function data(){
		$id = request()->input('id');
		if(!$id){
			return Json_Api(403,false,['msg' => '请求参数不足,缺少:id']);
		}
		if(!BlogArticle::query()->where('id',$id)->exists()){
			return Json_Api(403,false,['msg' => '文章不存在']);
		}
		return BlogArticle::query()->where('id',$id)->first();
	}
	
	#[PostMapping(path:"get")]
	public function get(){
		$id = request()->input('id');
		if(!$id){
			return Json_Api(403,false,['msg' => '请求参数不足,缺少:id']);
		}
		if(!Blog::query()->where('id',$id)->exists()){
			return Json_Api(403,false,['msg' => '博客不存在']);
		}
		return BlogArticle::query()->where('blog_id',$id)->paginate(request()->input('per_page',15));
	}
	
	#[PostMapping(path:"class")]
	public function page(){
		$id = request()->input('id');
		if(!$id){
			return Json_Api(403,false,['msg' => '请求参数不足,缺少:id']);
		}
		if(!Blog::query()->where('id',$id)->exists()){
			return Json_Api(403,false,['msg' => '博客不存在']);
		}
		return BlogClass::query()->where('blog_id',$id)->paginate(request()->input('per_page',15));
	}
	
	//#[PostMapping('classArticle')]
	/**public function classArticle(){
		$id = request()->input('id');
		if(!$id){
			return Json_Api(403,false,['msg' => '请求参数不足,缺少:id']);
		}
		$data = [];
		if(!BlogClass::query(true)->where('parent_id',$id)->exists()){
			$data[$id] = (array)BlogArticle::query(true)->where('class_id',$id)->get();
		}else{
			$ids = [];
			foreach(BlogClass::query(true)->where('parent_id',$id)->get() as $value){
				$ids[]=$value->id;
			}
			foreach($ids as $key=>$value){
				$data[$value] = (array)BlogArticle::query(true)->where('class_id',$value)->get();
			}
			$data[$id] = (array)BlogArticle::query(true)->where('class_id',$id)->get();
		}
		return $data;
	}
	 **/
}