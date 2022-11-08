<?php

namespace App\Plugins\Core\src\Controller\Pay;

use App\Middleware\AdminMiddleware;
use App\Plugins\Core\src\Handler\FileUpload;
use App\Plugins\Core\src\Models\PayConfig;
use Hyperf\HttpServer\Annotation\{Controller, GetMapping, Middleware, PostMapping};

#[Controller(prefix: '/admin/Pay')]
#[Middleware(AdminMiddleware::class)]
class PayAdminController
{
    /**
     * 订单
     * @return \Psr\Http\Message\ResponseInterface
     */
    #[GetMapping(path:"")]
    public function index(){
        return view('App::Pay.admin.index');
    }

    /**
     * 配置
     */
    #[GetMapping(path:"config")]
    public function config(){
        return view('App::Pay.admin.config');
    }

    #[PostMapping(path:'config')]
    public function config_save(FileUpload $fileUpload){
        // 先处理上传的文件
        foreach (request()->getUploadedFiles() as $name=>$file){
            // 获取文件
            $file = request()->file($name);
            if ($file->isFile()){
                // 保存文件
                $file = $fileUpload->save($file,'admin','pay_wechat_');
                // 获取文件路径
                $file = $file['raw_path'];
                // 插入到数据库
                $this->config_save_insert($name,$file);
            }
        }
        // 后处理保存的字符串内容
        foreach(request()->all() as $key => $value){
            $this->config_save_insert($key,$value);
        }
        pay()->clean_options();
        return redirect()->url('/admin/Pay/config')->with('success','更新成功')->go();
    }

    private function config_save_insert($name,$value): bool
    {
        if(PayConfig::query()->where('name',$name)->exists()){
            // 存在则更新
            PayConfig::query()->where('name',$name)->update([
                'value' => $value
            ]);
            return true;
        }

        // 不存在则新建
        PayConfig::query()->create([
            'value' => $value,
            'name' => $name
        ]);
        return true;
    }

    /**
     * 支付设置
     * @return \Psr\Http\Message\ResponseInterface
     */
    #[GetMapping(path:"setting")]
    public function setting(){
        return view('App::Pay.admin.setting');
    }

    #[PostMapping(path:"setting")]
    public function setting_submit(){
        $pay = request()->input('pay',[]);
        $pay = json_encode($pay,JSON_UNESCAPED_UNICODE);
        $this->config_save_insert('enable',$pay);
        pay()->clean_options();
        return redirect()->url('/admin/Pay/setting')->with('success','更新成功')->go();
        //return view('App::Pay.admin.setting');
    }
}