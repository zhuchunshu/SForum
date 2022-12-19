<?php
namespace App\Plugins\User\src\Controller\Admin;

use App\Middleware\AdminMiddleware;
use App\Plugins\User\src\Models\User;
use App\Plugins\User\src\Request\Create;
use App\Plugins\User\src\Request\Update;
use Hyperf\HttpServer\Annotation\Controller;
use Hyperf\HttpServer\Annotation\GetMapping;
use Hyperf\HttpServer\Annotation\Middleware;
use Hyperf\HttpServer\Annotation\PostMapping;
use App\Plugins\User\src\Models\UserClass as Uc;

#[Controller]
#[Middleware(\App\Middleware\AdminMiddleware::class)]
class UserClass
{
    #[GetMapping(path:"/admin/userClass")]
    public function index(){
        $page = \App\Plugins\User\src\Models\UserClass::query()->paginate(15);
        return view("User::Class.index",['page' => $page]);
    }

    /**
     * @GetMapping(path="/admin/userClass/create")
     */
    public function create(){
        return view("User::Class.create");
    }

    /**
     * @PostMapping(path="/admin/userClass/create")
     */
    public function create_post(Create $request): array
    {
        if(!admin_auth()->check()){
            return Json_Api(419,false,['msg' => '无权限']);
        }
        $data = $request->validated();
        $result = [
            "icon" => $data['icon'],
            "name" => $data['name'],
            "color" => $data['color'],
            "quanxian" => json_encode($data['quanxian'],JSON_UNESCAPED_UNICODE),
            'permission-value' => $data['permission-value']
        ];
        \App\Plugins\User\src\Models\UserClass::query()->create($result);
        return Json_Api(200,true,['msg' => '用户组创建成功!']);
    }

    /**
     * @GetMapping(path="/admin/userClass/edit/{id}")
     */
    public function edit($id){
        if(!\App\Plugins\User\src\Models\UserClass::query()->where("id",$id)->count()){
            return admin_abort(['msg' => 'id为'.$id.'的用户组不存在'],404);
        }
        return view("User::Class.edit",['id' => $id]);
    }

    /**
     * @PostMapping(path="/admin/userClass/{id}/data")
     */
    public function data($id){
        if(!admin_auth()->check()){
            return Json_Api(419,false,['msg' => '无权限']);
        }
        if(!\App\Plugins\User\src\Models\UserClass::query()->where("id",$id)->count()){
            return Json_Api(404,false,['msg' => 'id为'.$id.'的用户组不存在']);
        }
        $data = \App\Plugins\User\src\Models\UserClass::query()->where("id",$id)->first();
        $result = [
            "icon" => $data->icon,
            "name" => $data->name,
            "color" => $data->color,
            "quanxian" => json_decode($data->quanxian),
            'permission_value' => $data['permission-value']
        ];
        return Json_Api(200,true,$result);
    }

    /**
     * @PostMapping(path="/admin/userClass/update")
     */
    public function update(Update $request): array
    {
        if(!admin_auth()->check()){
            return Json_Api(419,false,['msg' => '无权限']);
        }
      $data = $request->validated();
      \App\Plugins\User\src\Models\UserClass::query()->where("id",$data['id'])->update([
         "icon" => $data['icon'],
         "name" => $data['name'],
         "color" => $data['color'],
         "quanxian" => json_encode($data['quanxian'],JSON_UNESCAPED_UNICODE),
          'permission-value' => $data['permission-value']
      ]);
      return Json_Api(200,true,['msg' => '更新成功!']);
    }

    #[PostMapping(path:"/admin/userClass/remove")]
    public function remove(){
        if(!admin_auth()->check()){
            return Json_Api(419,false,['msg' => '无权限']);
        }
        $id = request()->input('id');
        if(!is_numeric($id)){
            return Json_Api(403,false,['msg' => '被删除的用户组id有误']);
        }
        if($id ===1){
            return Json_Api(403,false,['msg' => '安全起见,你无法删除id为1的用户组,因为这是默认用户组']);
        }
        if(!Uc::query()->where("id",$id)->count()){
            return Json_Api(403,false,['msg' => 'id为'.$id.'的用户组不存在']);
        }
        Uc::query()->where("id",$id)->delete();

        // 把此用户组下的用户转移到id为1的用户组下
        go(function() use($id){
            User::query()->where('class_id', $id)->update(['class_id' => 1]);
        });
        return Json_Api(200,true,['msg' => '删除成功!']);
    }

    #[PostMapping(path:"/admin/userClass/Default/Authority")]
    public function Get_All_Default_Authority(){
        if(!admin_auth()->check()){
            return Json_Api(419,false,['msg' => '无权限']);
        }
        $arr = [];
        foreach(Itf()->get("core_auth_selected") as $value){
            $arr[]=$value;
        }
        return $arr;
    }

}