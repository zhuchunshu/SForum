<?php

namespace App\Plugins\Blog\src\Controller\Api;

use App\Plugins\Blog\src\Models\{Blog,BlogArticle,BlogClass};
use App\Plugins\Blog\src\Request\{ArticleCreate,ArticleEdit};
use App\Plugins\User\src\Middleware\LoginMiddleware;
use Hyperf\HttpServer\Annotation\{Controller,Middleware,PostMapping};
use Hyperf\Utils\Arr;

#[Controller(prefix:"/api/Blog/article")]
#[Middleware(LoginMiddleware::class)]
class Article
{
	#[PostMapping(path:"class")]
	public function classList(){
		if(!Authority()->check('blog')){
			return admin_abort('无权限',401);
		}
		$blog_id = Blog::query(true)->where('user_id',auth()->id())->exists();
		if(!$blog_id){
			return admin_abort('用户未开通博客',403);
		}
		$blog_id = Blog::query(true)->where('user_id',auth()->id())->value('id');
		$all = BlogClass::query()->where('blog_id',$blog_id)->get();
		$data = [];
		if(!count($all)){
			$data = Arr::add($data,0,[
				"text"=>'请先去创建分类',
				"value" => 'none',
				//"icons" => "&lt;span class=&quot;avatar avatar-xs&quot; style=&quot;background-image: url($value->icon)&quot;&gt;&lt;/span&gt;"
			]);
		}else{
			$data=Arr::add($data,0,[
				"text"=>'请选择分类',
				"value" => 'none',
				//"icons" => "&lt;span class=&quot;avatar avatar-xs&quot; style=&quot;background-image: url($value->icon)&quot;&gt;&lt;/span&gt;"
			]);
		}
		foreach($all as $key=>$value){
			$key++;
			$data=Arr::add($data,$key,[
				"text"=>$value->name,
				"value" => $value->id,
				//"icons" => "&lt;span class=&quot;avatar avatar-xs&quot; style=&quot;background-image: url($value->icon)&quot;&gt;&lt;/span&gt;"
			]);
		}
		return $data;
	}
	
	#[PostMapping(path:"create")]
	public function create(ArticleCreate $request){
		if(!Authority()->check('blog')){
			return Json_Api(401,false,['无权限']);
		}
		$blog_id = Blog::query(true)->where('user_id',auth()->id())->exists();
		if(!$blog_id){
			return Json_Api(403,false,['用户未开通博客']);
		}
		$blog_id = Blog::query(true)->where('user_id',auth()->id())->value('id');
		// 验证通过的数据
		$validated = $request->validated();
		// 判断是否可以使用分类
		if(!BlogClass::query()->where(['id' => $validated['class_id'],'blog_id' => $blog_id])->exists()){
			return Json_Api(401,false,['无权使用此分类']);
		}
		BlogArticle::query()->create([
			'blog_id' => $blog_id,
			'title' => $validated['title'],
			'class_id' => $validated['class_id'],
			'content' => $validated['content'],
			'markdown' => $validated['markdown']
		]);
		session()->flash('success', '发布成功!');
		return Json_Api(200,true,['msg'=>'/blog/'.auth()->data()->username.".html"]);
	}
	
	#[PostMapping(path:"edit")]
	public function edit(ArticleEdit $request){
		if(!Authority()->check('blog')){
			return Json_Api(401,false,['无权限']);
		}
		$blog_id = Blog::query(true)->where('user_id',auth()->id())->exists();
		if(!$blog_id){
			return Json_Api(403,false,['用户未开通博客']);
		}
		$blog_id = Blog::query(true)->where('user_id',auth()->id())->value('id');
		
		// 验证通过的数据
		$validated = $request->validated();
		
		if(!BlogArticle::query()->where(['id' => $validated['id'],'blog_id' => $blog_id])->exists()){
			return Json_Api(401,false,['无权限']);
		}
		// 判断是否可以使用分类
		if(!BlogClass::query()->where(['id' => $validated['class_id'],'blog_id' => $blog_id])->exists()){
			return Json_Api(401,false,['无权使用此分类']);
		}
		BlogArticle::query()->where('id',$validated['id'])->update([
			'title' => $validated['title'],
			'class_id' => $validated['class_id'],
			'content' => $validated['content'],
			'markdown' => $validated['markdown']
		]);
		session()->flash('success', '更新成功!');
		return Json_Api(200,true,['msg'=>'/blog/article/'.$validated['id'].".html"]);
	}
}