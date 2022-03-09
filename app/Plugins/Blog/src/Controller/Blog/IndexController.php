<?php

namespace App\Plugins\Blog\src\Controller\Blog;

use App\Plugins\Blog\src\Models\Blog;
use App\Plugins\Blog\src\Models\BlogArticle;
use App\Plugins\User\src\Middleware\LoginMiddleware;
use App\Plugins\User\src\Models\User;
use Hyperf\HttpServer\Annotation\Controller;
use Hyperf\HttpServer\Annotation\GetMapping;
use Hyperf\HttpServer\Annotation\Middleware;
use Hyperf\Utils\Str;

#[Controller(prefix:"/blog")]
class IndexController
{
	#[GetMapping(path:"")]
	public function index(){
		$created = Blog::query()->where('user_id',auth()->id())->exists();
		if($created){
			return redirect()->url('/blog/'.auth()->data()->username.".html")->go();
		}
		return view('Blog::index',['created' => $created]);
	}
	
	#[GetMapping(path:"create")]
	#[Middleware(LoginMiddleware::class)]
	public function create(){
		if(!Authority()->check('blog')){
			return admin_abort('无权限',401);
		}
		$created = Blog::query()->where('user_id',auth()->id())->exists();
		if($created===true){
			return redirect()->url('/blog/'.auth()->data()->username.".html")->with('info','无需重复创建博客')->go();
		}
		Blog::query()->create([
			'user_id' => auth()->id(),
			'token' => Str::random()
		]);
		return redirect()->url('/blog/'.auth()->data()->username.".html")->with('success','创建成功!')->go();
	}
	// 我的博客
	#[GetMapping(path:"{username}.html")]
	public function myBlog($username){
		if(!Authority()->check('blog')){
			return admin_abort('无权限',401);
		}
		$username = urldecode($username);
		if(!User::query(true)->where('username',$username)->exists()){
			return admin_abort('用户:'.$username.'不存在',403);
		}
		$user = User::query(true)->where('username',$username)->first();
		$blog_id = Blog::query(true)->where('user_id',$user->id)->value('id');
		$page = BlogArticle::query(true)->where('blog_id',$blog_id)->orderByDesc('created_at')->paginate(15);
		
		return view('Blog::blog.index',['user'=>$user,'page' => $page]);
	}
}