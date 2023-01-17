<?php

declare(strict_types=1);
/**
 * This file is part of zhuchunshu.
 * @link     https://github.com/zhuchunshu
 * @document https://github.com/zhuchunshu/super-forum
 * @contact  laravel@88.com
 * @license  https://github.com/zhuchunshu/super-forum/blob/master/LICENSE
 */
namespace App\Plugins\QQPusher\src\Controller;

use App\Middleware\AdminMiddleware;
use App\Model\AdminOption;
use App\Plugins\QQPusher\src\GoCqhttp;
use App\Plugins\QQPusher\src\Models\QQpusher;
use App\Plugins\Topic\src\Models\Topic;
use Hyperf\HttpServer\Annotation\Controller;
use Hyperf\HttpServer\Annotation\GetMapping;
use Hyperf\HttpServer\Annotation\Middleware;
use Hyperf\HttpServer\Annotation\PostMapping;
use Noodlehaus\Config;

#[Controller(prefix: '/admin/QQPusher')]
#[Middleware(AdminMiddleware::class)]
class AdminController
{
    #[GetMapping(path: '')]
    public function index()
    {
        $response = GoCqhttp::post('get_group_list');
        if ($response['status'] === 'ok') {
            $groups = $response['data'];
        } else {
            $groups = [];
        }
        return view('QQPusher::admin.index', ['groups' => $groups]);
    }

    #[PostMapping(path: 'save')]
    public function save()
    {
        $url = request()->input('QQPusher_POST_URL', 'http://127.0.0.1:5700');
        AdminOption::query()->updateOrInsert(['name' => 'QQPusher_POST_URL'], ['value' => $url]);
        options_clear();
        return redirect()->url('/admin/QQPusher')->with('success', '修改成功!')->go();
    }

    #[GetMapping(path: 'groups')]
    public function get_groups()
    {
        if (! file_exists(plugin_path('QQPusher/groups.json'))) {
            return [];
        }

        return Json_Api(200, true, ['msg' => '查询成功!', 'data' => Config::load(plugin_path('QQPusher/groups.json'))->all()]);
    }

    #[PostMapping(path: 'groups')]
    public function set_groups()
    {
        $data = request()->input('data');
        if (! is_array($data)) {
            $data = [];
        }
        $data = json_encode($data);
        file_put_contents(plugin_path('QQPusher/groups.json'), $data);
        return Json_Api(200, true, ['msg' => '更新成功!']);
    }
}
