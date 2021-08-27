<?php


namespace App\Plugins\User\src\Controller;

use App\Plugins\Core\src\Handler\UploadHandler;
use Hyperf\HttpServer\Annotation\Controller;
use Hyperf\HttpServer\Annotation\PostMapping;

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
}