<?php


namespace App\Plugins\Topic\src\Controllers;

use App\Plugins\Core\src\Handler\AvatarUpload;
use App\Plugins\Topic\src\Models\Topic;
use App\Plugins\Topic\src\Models\TopicTag;
use App\Plugins\Topic\src\Requests\CreateTagRequest;
use App\Plugins\Topic\src\Requests\EditTagRequest;
use Hyperf\HttpServer\Annotation\Controller;
use Hyperf\HttpServer\Annotation\GetMapping;
use Hyperf\HttpServer\Annotation\PostMapping;
use Hyperf\Utils\Str;

#[Controller]
class TagsController
{
    #[GetMapping(path: "/tags")]
    public function index(): \Psr\Http\Message\ResponseInterface
    {
        $page = TopicTag::query()->paginate(15,['*'],"TagPage");
        return view("Topic::Tags.index",['page' => $page]);
    }

    #[GetMapping(path: "/tags/{id}.html")]
    public function data($id){
        if(!TopicTag::query()->where('id',$id)->exists()) {
            return admin_abort("页面不存在",404);
        }
        $page = Topic::query()
            ->where("tag_id",$id)
            ->with("tag","user")
            ->orderBy("id","desc")
            ->paginate(get_options("topic_home_num",15));
        $data = TopicTag::query()->where("id",$id)->first();
        return view("Topic::Tags.data",['data' => $data,'page' => $page]);
    }
	
	#[GetMapping(path:"/tags/create")]
	public function create(){
		if(!auth()->check() || !Authority()->check('topic_tag_create')){
			return admin_abort('权限不足',401);
		}
		$userClass = \App\Plugins\User\src\Models\UserClass::query()->get();
		return view("Topic::Tags.create",['userClass' => $userClass]);
	}
	
	#[PostMapping(path:"/tags/create")]
	public function create_store(CreateTagRequest $request,AvatarUpload $upload){
		if(!auth()->check() || !Authority()->check('topic_tag_create')){
			return Json_Api(401,false,['msg' => '无权限']);
		}
		$name = $request->input("name");
		$color = $request->input("color");
		$description = $request->input("description");
		$icon = $request->file("icon");
		$userClass = $request->input("userClass");
		if($userClass){
			$userClass = json_encode($userClass, JSON_THROW_ON_ERROR,JSON_UNESCAPED_UNICODE);
		}else{
			$userClass = null;
		}
		$icon_url = $upload->save($icon,"admin",Str::random(),200)['path'];
		TopicTag::create([
			"name" => $name,
			"color" => $color,
			"description" => $description,
			"icon" => $icon_url,
			'userClass' => $userClass,
			"user_id" => auth()->id()
		]);
		return redirect()->url("/tags/create")->with("success","创建成功!")->go();
	}
	
	#[GetMapping(path:"/tags/{id}/edit")]
	public function edit($id){
		if(!auth()->check() || !Authority()->check('topic_tag_create')){
			return admin_abort('权限不足',401);
		}
		if(!TopicTag::query()->where("id",$id)->count()){
			return admin_abort('id为'.$id.'的标签不存在',403);
		}
		$data = TopicTag::query()->find($id);
		if((int)$data->user_id!== (int)auth()->id()){
			return admin_abort('您无权限修改',401);
		}
		$userClass = \App\Plugins\User\src\Models\UserClass::query()->get();
		return view("Topic::Tags.edit",['data' => $data,'userClass' => $userClass]);
	}
	
	#[PostMapping(path:"/tags/edit")]
	public function edit_post(EditTagRequest $request,AvatarUpload $upload){
		if(!auth()->check() || !Authority()->check('topic_tag_create')){
			return admin_abort('权限不足',401);
		}
		$icon = false;
		$id = $request->input("id");
		$name = $request->input("name");
		$description = $request->input("description");
		$color = $request->input("color");
		if($request->hasFile("icon")){
			$icon = true;
		}
		$userClass = $request->input("userClass");
		if($userClass){
			$userClass = json_encode($userClass, JSON_THROW_ON_ERROR,JSON_UNESCAPED_UNICODE);
		}else{
			$userClass = null;
		}
		
		$data = TopicTag::query()->find($id);
		if((int)$data->user_id!== (int)auth()->id()){
			return admin_abort('您无权限修改',401);
		}
		
		if($icon===true){
			// 上传icon
			$url = $upload->save($request->file("icon"),"admin",Str::random(),200)['path'];
			TopicTag::query()->where("id",$id)->update([
				"name" => $name,
				"description" => $description,
				"color" => $color,
				"icon" => $url,
				'userClass' => $userClass
			]);
		}else{
			TopicTag::query()->where("id",$id)->update([
				"name" => $name,
				"description" => $description,
				"color" => $color,
				'userClass' => $userClass
			]);
		}
		return redirect()->url('/tags/'.$id.'/edit')->with("success","修改成功!")->go();
		
	}
}