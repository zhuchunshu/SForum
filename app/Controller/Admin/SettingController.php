<?php

declare(strict_types=1);

namespace App\Controller\Admin;

use App\Model\AdminOption;
use App\Model\AdminUser;
use App\CodeFec\Admin\Ui;
use App\CodeFec\Admin\Admin;
use App\CodeFec\Admin\Ui\Card;
use Hyperf\Di\Annotation\Inject;
use Hyperf\HttpServer\Annotation\Controller;
use Hyperf\HttpServer\Annotation\GetMapping;
use Hyperf\HttpServer\Annotation\Middleware;
use Hyperf\HttpServer\Annotation\PostMapping;
use Hyperf\HttpServer\Annotation\RequestMapping;
use Hyperf\HttpServer\Contract\RequestInterface;
use Hyperf\HttpServer\Contract\ResponseInterface;
use Hyperf\Validation\Contract\ValidatorFactoryInterface;
use HyperfExt\Hashing\Hash;

/**
 * Class SettingController
 * @Controller()
 * @Middleware(\App\Middleware\AdminMiddleware::class)
 * @package App\Controller\Admin
 */
class SettingController
{
    /**
     * @Inject()
     * @var ValidatorFactoryInterface
     */
    protected $validationFactory;

    /**
     * 个人设置
     * @RequestMapping(path="im", methods="get")
     */
    public function im(Ui $ui, Card $card): \Psr\Http\Message\ResponseInterface
    {
        return $ui
            ->title("个人设置")
            ->ImportJs([mix("js/admin/setting.js")])
            ->body($card->title("个人设置")
                ->content(view("admin.setting.im"))
                ->render())
            ->render();
    }
    /**
     * 个人设置
     * @RequestMapping(path="im", methods="post")
     */
    public function imPost(): array
    {
        $type = request()->input("type");
        switch ($type) {
            case 'username':
                if(!request()->input("username")){
                    return Json_Api(403,false,["msg" => "用户名格式不能为空"]);
                }
                AdminUser::query()->where("id",Admin::id())->update([
                    "username" => request()->input("username")
                ]);
                $data = AdminUser::query()->where('id', Admin::id())->first();
                session()->set('admin', $data);
                return Json_Api(200,true,["msg" => "修改成功!"]);
            case 'email':
                $v = $this->validationFactory->make(
                    request()->all(),
                    [
                        'email' => 'required|email'
                    ]
                )->validated();
                
                AdminUser::query()->where("id",Admin::id())->update([
                    "email" => $v['email']
                ]);
                $data = AdminUser::query()->where('id', Admin::id())->first();
                session()->set('admin', $data);
                return Json_Api(200,true,["msg" => "修改成功!"]);
                break;
            case 'password':
                $v = $this->validationFactory->make(
                    request()->all(),
                    [
                        'old_pwd' => 'required',
                        'new_pwd' => 'required|min:6'
                    ],
                    [],
                    ['old_pwd'=>"旧密码",'new_pwd'=>"新密码"]
                )->validated();
                if(!Hash::check($v['old_pwd'], AdminUser::query()->where("id",Admin::id())->first()->password)){
                    return Json_Api(403,false,["旧密码错误"]);
                }
                AdminUser::query()->where("id",Admin::id())->update([
                    "password" => Hash::make($v['new_pwd'])
                ]);
                $data = AdminUser::query()->where('id', Admin::id())->first();
                session()->set('admin', $data);
                return Json_Api(200,true,["msg" => "修改成功!"]);
                break;

            default:
                return Json_Api(403, false, ['msg' => '请求方法不存在']);
                break;
        }
    }

    /**
     * @GetMapping(path="/admin/setting")
     */
    public function setting(){
        return view("admin.setting.core");
    }

    /**
     * @PostMapping(path="/admin/setting")
     */
    public function setting_post(): array
    {
        if(!admin_auth()->check()){
            return Json_Api(401,false,['msg' => '无权限']);
        }
        if(!is_array(request()->input('data'))){
            $data = de_stringify(request()->input('data'));
        }else{
            $data = request()->input('data');
        }

        if(!is_array($data)){
            return Json_Api(403,false,['msg' => '请提交正确的数据']);
        }

        foreach ($data as $key=>$value){
            $name = ['name' => $key];
            $values = ['value' => $value];
            AdminOption::query()->updateOrInsert($name,$values);
        }
        if(request()->input('env')){
            $env = de_stringify(request()->input('env'));
            if(!is_array($env)){
                return Json_Api(403,false,['msg' => '请提交正确的数据']);
            }
            $env_arr = [];
            foreach ($env as $key=>$value){
                if($key && $value && is_string($key) && $key !== "=" && is_string($value) && $value!="="){
                    $env_arr[$key] = $value;
                }
            }
            modifyEnv($env_arr);
        }
	    options_clear();
        return Json_Api(200,true,['msg' => '更新成功!']);
    }
	
	#[PostMapping(path:"/admin/setting/clearCache")]
	public function setting_clearCache(){
		options_clear();
		return Json_Api(200,true,['msg' => '清理成功!']);
	}

    /**
     * @PostMapping(path="/api/adminOptionList")
     */
    public function adminOptionList(): array
    {
        $result = [];
        foreach (AdminOption::query()->select('name', 'value')->get() as $value){
            $result[$value->name]=$value->value;
        }
        return Json_Api(200,true,$result);
    }

    /**
     * @PostMapping(path="/api/adminEnvList")
     */
    public function adminEnvList(): array
    {
        $result = [];
        $content = file_get_contents(BASE_PATH."/.env");
        foreach (explode("\n",$content) as $value) {
            if($value && is_string($value)){
                $arr = explode("=",$value);
                $result[$arr[0]]=$arr[1];
            }
        }
        return Json_Api(200,true,$result);
    }
}
