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
}