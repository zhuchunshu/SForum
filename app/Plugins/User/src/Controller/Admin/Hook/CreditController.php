<?php

declare (strict_types=1);
/**
 * This file is part of zhuchunshu.
 * @link     https://github.com/zhuchunshu
 * @document https://github.com/zhuchunshu/SForum
 * @contact  laravel@88.com
 * @license  https://github.com/zhuchunshu/SForum/blob/master/LICENSE
 */
namespace App\Plugins\User\src\Controller\Admin\Hook;

use Hyperf\HttpServer\Annotation\Controller;
use Hyperf\HttpServer\Annotation\GetMapping;
use Hyperf\HttpServer\Annotation\PostMapping;
#[Controller(prefix: '/admin/hook/credits')]
class CreditController
{
    #[GetMapping('')]
    public function index()
    {
        return view('User::Admin.hook.credit');
    }
    #[PostMapping('')]
    public function store() : \Psr\Http\Message\ResponseInterface
    {
        $data = request()->all();
        if (arr_has($data, '_token')) {
            unset($data['_token']);
        }
        foreach ($data as $k => $v) {
            go(function () use($k, $v) {
                set_options('hook_user_credit_' . $k, $v);
            });
        }
        return redirect()->back()->with('success', '保存成功')->go();
    }
}