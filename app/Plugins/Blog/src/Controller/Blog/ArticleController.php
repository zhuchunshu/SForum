<?php

namespace App\Plugins\Blog\src\Controller\Blog;

use App\Plugins\Blog\src\Models\Blog;
use App\Plugins\Blog\src\Models\BlogArticle;
use App\Plugins\Blog\src\Models\BlogClass;
use App\Plugins\User\src\Middleware\LoginMiddleware;
use Hyperf\HttpServer\Annotation\Controller;
use Hyperf\HttpServer\Annotation\GetMapping;
use Hyperf\HttpServer\Annotation\Middleware;

#[Controller(prefix:"/blog/article")]
class ArticleController
{
	#[GetMapping(path:"create")]
	#[Middleware(LoginMiddleware::class)]
	public function create(){
		if(!Authority()->check('blog')){
			return admin_abort('无权限',401);
		}
		$blog_id = Blog::query(true)->where('user_id',auth()->id())->exists();
		if(!$blog_id){
			return admin_abort('用户未开通博客',403);
		}
		$blog_id = Blog::query(true)->where('user_id',auth()->id())->value('id');
		$class = BlogClass::query()->where('blog_id',$blog_id)->get();
		return view('Blog::blog.article.create',['class' =>$class]);
	}
	
	#[GetMapping("{id}.html")]
	public function show($id){
		if(!BlogArticle::query()->where('id',$id)->exists()){
			return admin_abort('页面不存在',404);
		}
		$blog_id = BlogArticle::query()->where('id',$id)->value('blog_id');
		$user = Blog::query()->with('user')->where('id',$blog_id)->first()->user;
		$data = BlogArticle::query()->where('id',$id)->first();
		$quanxian = false;
		if(Authority()->check('blog') && (int)$user->id===(int)auth()->id()){
			$quanxian = true;
		}
		
		$blog = Blog::query(true)->where('id',$blog_id)->first();
		return view("Blog::blog.article.show",['user' =>$user,'data' => $data,'quanxian' => $quanxian,'blog' => $blog]);
	}
	
	// 删除文章
	#[GetMapping(path:"{id}/remove")]
	public function remove($id){
		if(csrf_token()!==request()->input('_token')){
			return admin_abort('csrf验证失败',401);
		}
		if(!BlogArticle::query()->where('id',$id)->exists()){
			return admin_abort('页面不存在',404);
		}
		$blog_id = BlogArticle::query()->where('id',$id)->value('blog_id');
		$user = Blog::query()->with('user')->where('id',$blog_id)->first()->user;
		$quanxian = false;
		if(Authority()->check('blog') && (int)$user->id===(int)auth()->id()){
			$quanxian = true;
		}
		if($quanxian===true){
			BlogArticle::query()->where('id',$id)->delete();
			return redirect()->url('/blog/'.auth()->data()->username.".html")->with('success','删除成功!')->go();
		}
		return redirect()->url('/blog/'.auth()->data()->username.".html")->with('danger','删除失败!');
	}
	
	#[GetMapping(path:"{id}/edit")]
	public function edit($id){
		if(!BlogArticle::query()->where('id',$id)->exists()){
			return admin_abort('页面不存在',404);
		}
		$blog_id = BlogArticle::query()->where('id',$id)->value('blog_id');
		$user = Blog::query()->with('user')->where('id',$blog_id)->first()->user;
		if(!Authority()->check('blog') || (int)$user->id!==(int)auth()->id()){
			return admin_abort('权限不足!',401);
		}
		$data = BlogArticle::query()->where('id',$id)->first();
		return view('Blog::blog.article.edit',['data'=>$data]);
	}
}