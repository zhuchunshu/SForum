<?php


namespace App\Plugins\User\src\Controller;

use App\Plugins\Core\src\Handler\UploadHandler;
use App\Plugins\User\src\Models\User;
use Hyperf\HttpServer\Annotation\Controller;
use Hyperf\HttpServer\Annotation\GetMapping;
use Hyperf\HttpServer\Annotation\PostMapping;
use Hyperf\Utils\Arr;

#[Controller]
class ApiController
{
    #[PostMapping(path:"/user/upload/image")]
    public function up_image(UploadHandler $uploader){
        $data = [];
        foreach (request()->file('file') as $key => $file) {
            $result = $uploader->save($file, 'topic', auth()->id());
            if ($result) {
                $url = $result['path'];
                $data['data']['succMap'][$url]=$url;
            } else {
                array_push((array)$data['data']['errFiles'],$key);
            }
        }
        return $data;
    }
    #[PostMapping(path:"/api/user/@user_list")]
    public function user_list(): array
    {
        $data = User::query()->select('username','id')->get();
        $arr = [];
        foreach ($data as $key=>$value){
            $arr = Arr::add($arr,$key,["value"=>"@".$value->username,"html" => '<img src="'.avatar_url($value->id).'" alt="'.$value->username.'"/> '.$value->username]);
        }
        return $arr;
    }

    #[PostMapping(path:"/api/user/@has_user_username/{username}")]
    public function has_user_username($username):array{
        if(User::query()->where("username",$username)->count()){
            return Json_Api(200,true,['msg' => '此用户存在']);
        }
        return Json_Api(404, false,['msg' => '用户:'.$username.'不存在']);
    }

    #[PostMapping(path:"/api/user/get.user.avatar.url")]
    public function get_user_avatar_url(){
        $user_id = request()->input("user_id");
        if(!$user_id){
           return Json_Api(403,false,['请求参数不足,缺少:user_id']);
        }
        if(!User::query()->where("id",$user_id)->exists()){
            return Json_Api(403,false,['此用户不存在']);
        }
        $data = User::query()->where("id",$user_id)->first();
        return  Json_Api(200,true,['msg'=>super_avatar($data)]);
    }

    #[PostMapping(path:"/api/user/get.user.data")]
    public function get_user_data(){
        $user_id = request()->input("user_id");
        if(!$user_id){
            return Json_Api(403,false,['请求参数不足,缺少:user_id']);
        }
        if(!User::query()->where("id",$user_id)->exists()){
            return Json_Api(403,false,['此用户不存在']);
        }
        $data = User::query()
            ->where("id",$user_id)
            ->with("Class",'options')
            ->first();
        $data['avatar'] = super_avatar($data);
        $data['group'] = '<a href="/users/group/'.$data->class_id.'.html">'.Core_Ui()->Html()->UserGroup($data->Class).'</a>';
        return Json_Api(200,true,$data);
    }
}