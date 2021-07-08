<?php

declare(strict_types=1);

namespace App\Controller\Admin;

use App\Model\AdminUser;
use App\CodeFec\Admin\Ui;
use App\CodeFec\Admin\Admin;
use App\CodeFec\Admin\Ui\Card;
use Hyperf\Di\Annotation\Inject;
use Hyperf\HttpServer\Annotation\Controller;
use Hyperf\HttpServer\Annotation\Middleware;
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
}
