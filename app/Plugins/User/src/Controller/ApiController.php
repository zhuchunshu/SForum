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
}