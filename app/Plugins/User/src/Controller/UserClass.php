<?php
namespace App\Plugins\User\src\Controller;

use App\Plugins\User\src\Request\Create;
use App\Plugins\User\src\Request\Update;
use Hyperf\HttpServer\Annotation\Controller;
use Hyperf\HttpServer\Annotation\GetMapping;
use Hyperf\HttpServer\Annotation\Middleware;
use Hyperf\HttpServer\Annotation\PostMapping;

/**
 * Class UserClass
 * @Controller
 * @Middleware(\App\Middleware\AdminMiddleware::class)
 * @package App\Plugins\User\src\Controller
 */
class UserClass
{
    /**
     * @GetMapping(path="/admin/userClass")
     */
    public function index(){
        $page = \App\Plugins\User\src\Models\UserClass::query()->paginate(15);
        return view("plugins.User.Class.index",['page' => $page]);
    }

    /**
     * @GetMapping(path="/admin/userClass/create")
     */
    public function create(){
        return view("plugins.User.Class.create");
    }

    /**
     * @PostMapping(path="/admin/userClass/create")
     */
    public function create_post(Create $request): array
    {
        \App\Plugins\User\src\Models\UserClass::query()->create($request->validated());
        return Json_Api(200,true,['msg' => '用户组创建成功!']);
    }

    /**
     * @GetMapping(path="/admin/userClass/edit/{id}")
     */
    public function edit($id){
        if(!\App\Plugins\User\src\Models\UserClass::query()->where("id",$id)->count()){
            return admin_abort(['msg' => 'id为'.$id.'的用户组不存在'],404);
        }
        return view("plugins.User.Class.edit",['id' => $id]);
    }

    /**
     * @PostMapping(path="/admin/userClass/{id}/data")
     */
    public function data($id){
        if(!\App\Plugins\User\src\Models\UserClass::query()->where("id",$id)->count()){
            return Json_Api(404,false,['msg' => 'id为'.$id.'的用户组不存在']);
        }
        $data = \App\Plugins\User\src\Models\UserClass::query()->where("id",$id)->first();
        $result = [
            "icon" => $data->icon,
            "name" => $data->name,
            "color" => $data->color,
            "quanxian" => $data->quanxian
        ];
        return Json_Api(200,true,$result);
    }

    /**
     * @PostMapping(path="/admin/userClass/update")
     */
    public function update(Update $request): array
    {
      $data = $request->validated();
      \App\Plugins\User\src\Models\UserClass::query()->where("id",$data['id'])->update([
         "icon" => $data['icon'],
         "name" => $data['name'],
         "color" => $data['color'],
         "quanxian" => $data['quanxian']
      ]);
      return Json_Api(200,true,['msg' => '更新成功!']);
    }
}