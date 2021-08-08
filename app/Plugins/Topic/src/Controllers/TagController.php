<?php


namespace App\Plugins\Topic\src\Controllers;


use App\Middleware\AdminMiddleware;
use App\Plugins\Core\src\Handler\AvatarUpload;
use App\Plugins\Topic\src\Models\TopicTag;
use App\Plugins\Topic\src\Requests\CreateTagRequest;
use Hyperf\HttpServer\Annotation\Controller;
use Hyperf\HttpServer\Annotation\GetMapping;
use Hyperf\HttpServer\Annotation\Middleware;
use Hyperf\HttpServer\Annotation\PostMapping;
use Hyperf\Utils\Str;

#[Middleware(AdminMiddleware::class)]
#[Controller]
class TagController
{
    #[GetMapping(path:"/admin/topic/tag/create")]
    public function create(){
        return view("plugins.Topic.Tag.create");
    }

    #[PostMapping(path:"/admin/topic/tag/create")]
    public function create_store(CreateTagRequest $request,AvatarUpload $upload){
        $name = $request->input("name");
        $color = $request->input("color");
        $description = $request->input("description");
        $icon = $request->file("icon");
        $icon_url = $upload->save($icon,"admin",Str::random())['path'];
        TopicTag::create([
            "name" => $name,
            "color" => $color,
            "description" => $description,
            "icon" => $icon_url
        ]);
        return redirect()->back()->with("success","创建成功!")->go();
    }
}