<?php

namespace App\Plugins\Blog\src\Controller\Blog;

use App\Plugins\Blog\src\Models\Blog;
use App\Plugins\Blog\src\Models\BlogArticle;
use App\Plugins\Blog\src\Models\BlogClass;
use App\Plugins\Blog\src\Request\ClassEdit;
use App\Plugins\User\src\Middleware\LoginMiddleware;
use App\Plugins\User\src\Models\User;
use Hyperf\HttpServer\Annotation\Controller;
use Hyperf\HttpServer\Annotation\GetMapping;
use Hyperf\HttpServer\Annotation\Middleware;
use Hyperf\HttpServer\Annotation\PostMapping;
use Hyperf\Utils\Str;

#[Controller(prefix:"/blog/class")]
class ClassController
{
	
	// 分类管理
	#[Middleware(LoginMiddleware::class)]
	#[GetMapping(path:"")]
	public function _class(){
		if(!Authority()->check('blog')){
			return admin_abort('无权限',401);
		}
		$blog_id = Blog::query(true)->where('user_id',auth()->id())->exists();
		if(!$blog_id){
			return admin_abort('你未开通博客',403);
		}
		$blog_id = Blog::query(true)->where('user_id',auth()->id())->value('id');
		return view('Blog::blog.Class');
	}
	
	// 分类列表
	#[Middleware(LoginMiddleware::class)]
	#[GetMapping(path:"list")]
	public function class_list(){
		if(!Authority()->check('blog')){
			return admin_abort('无权限',401);
		}
		$blog_id = Blog::query(true)->where('user_id',auth()->id())->exists();
		if(!$blog_id){
			return admin_abort('你未开通博客',403);
		}
		$blog_id = Blog::query(true)->where('user_id',auth()->id())->value('id');
		if(request()->input('parent_id')){
			$classList = BlogClass::query()->where([['blog_id',$blog_id],['parent_id', request()->input('parent_id')]])->paginate(15);
		}else{
			$classList = BlogClass::query()->where('blog_id',$blog_id)->paginate(15);
		}
		return view('Blog::blog.ClassList',['list'=>$classList]);
	}
	
	// 创建分类视图
	#[Middleware(LoginMiddleware::class)]
	#[GetMapping(path:"create")]
	public function class_create(){
		if(!Authority()->check('blog')){
			return admin_abort('无权限',401);
		}
		$blog_id = Blog::query(true)->where('user_id',auth()->id())->exists();
		if(!$blog_id){
			return admin_abort('你未开通博客',403);
		}
		$blog_id = Blog::query(true)->where('user_id',auth()->id())->value('id');
		$classList = BlogClass::query()->where('blog_id',$blog_id)->get();
		return view('Blog::blog.ClassCreate',['classList' => $classList]);
	}
	
	
	// 创建分类
	#[Middleware(LoginMiddleware::class)]
	#[PostMapping(path:"create")]
	public function class_create_submit(){
		if(!Authority()->check('blog')){
			return admin_abort('无权限',401);
		}
		$blog_id = Blog::query(true)->where('user_id',auth()->id())->exists();
		if(!$blog_id){
			return redirect()->with('danger','你未开通博客')->back()->go();
		}
		
		// 从数据库读到的数据
		$blog_id = Blog::query(true)->where('user_id',auth()->id())->value('id');
		
		// 接收到的数据
		$class_id = request()->input('class_id');
		$name = request()->input('name');
		if(!$class_id || !$name){
			return redirect()->with('danger','请求参数不足!')->url('/blog/class')->go();
		}
		
		if(BlogClass::query()->where(['blog_id' => $blog_id,'name' => $name])->exists()){
			return redirect()->with('danger','分类已存在,换个名字吧!')->url('/blog/class')->go();
		}
		
		if(!BlogClass::query()->where(['blog_id' => $blog_id,'id' => $class_id])->exists()){
			$class_id = null;
		}
		
		BlogClass::query()->create([
			'token' => Str::random(),
			'blog_id' => $blog_id,
			'parent_id' => $class_id,
			'name' => $name
		]);
		
		return redirect()->with('success','创建成功')->url('/blog/class')->go();
	}
	
	// 删除分类
	#[PostMapping(path:"remove")]
	#[Middleware(LoginMiddleware::class)]
	public function removeClass(): array|\Psr\Http\Message\ResponseInterface
	{
		if(!Authority()->check('blog')){
			return admin_abort('无权限',401);
		}
		$token = request()->input('token');
		if(!$token){
			return Json_Api(403,false,['msg' => '请求参数不足!']);
		}
		$blog_id = Blog::query(true)->where('user_id',auth()->id())->exists();
		if(!$blog_id){
			return Json_Api(403,false,['msg' => '用户未开通博客!']);
		}
		$blog_id = Blog::query(true)->where('user_id',auth()->id())->value('id');
		if(BlogClass::query()->where(['blog_id' => $blog_id,'token' =>$token])->delete()){
			return Json_Api(200,true,['msg' => '删除成功!']);
		}
		return Json_Api(200,true,['msg' => '删除失败!']);
	}
	
	// 修改分类
	#[GetMapping('{token}/edit')]
	#[Middleware(LoginMiddleware::class)]
	public function classEdit($token){
		if(!Authority()->check('blog')){
			return admin_abort('无权限',401);
		}
		$blog_id = Blog::query(true)->where('user_id',auth()->id())->exists();
		if(!$blog_id){
			return admin_abort('用户未开通博客');
		}
		if(!BlogClass::query()->where(['blog_id' => $blog_id,'token' =>$token])->exists()){
			return admin_abort('页面不存在',404);
		}
		$classList = BlogClass::query()->where('blog_id',$blog_id)->get();
		$data = BlogClass::query()->where(['blog_id' => $blog_id,'token' =>$token])->first();
		return view("Blog::blog.ClassEdit",['data' => $data,'classList' => $classList]);
	}
	
	// 修改分类-submit
	#[PostMapping(path:"edit")]
	public function classEditSubmit(ClassEdit $request){
		$data =  $request->validated();
		$blog_id = Blog::query(true)->where('user_id',auth()->id())->exists();
		if(!$blog_id){
			return Json_Api(401,false,['msg' => '用户未开通博客']);
		}
		if(!BlogClass::query()->where(['blog_id' => $blog_id,'token' =>$data['token']])->exists()){
			return Json_Api(403,false,['msg' => '无权限']);
		}
		$_data = BlogClass::query()->where(['blog_id' => $blog_id,'token' =>$data['token']])->first();
		$parent_id = $data['class_id'];
		$name = $data['name'];
		if($name!==$_data->name){
			if(BlogClass::query()->where(['blog_id' => $blog_id,'name' => $name])->exists()){
				return Json_Api(403,false,['msg' => '名称被占用']);
			}
		}
		//(int)$value->parent_id!==(int)$data->id
		if(!BlogClass::query()->where(['blog_id' => $blog_id,'id' => $parent_id])->exists() && BlogClass::query()->where(['blog_id' => $blog_id,'parent_id' => $data->id])->exists()){
			$parent_id = null;
		}
		BlogClass::query()->where(['blog_id' => $blog_id,'token' =>$data['token']])->update([
			'name' => $name,
			'parent_id' => $parent_id
		]);
		return redirect()->back()->with('success','更新成功!')->go();
	}
}