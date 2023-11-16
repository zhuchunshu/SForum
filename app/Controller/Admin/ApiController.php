<?php

declare(strict_types=1);
/**
 * This file is part of zhuchunshu.
 * @link     https://github.com/zhuchunshu
 * @document https://github.com/zhuchunshu/SForum
 * @contact  laravel@88.com
 * @license  https://github.com/zhuchunshu/SForum/blob/master/LICENSE
 */
namespace App\Controller\Admin;

use App\Jobs\Upgrading;
use App\Middleware\AdminMiddleware;
use App\Plugins\Topic\src\ContentParse;
use Hyperf\Di\Annotation\Inject;
use Hyperf\HttpServer\Annotation\Controller;
use Hyperf\HttpServer\Annotation\GetMapping;
use Hyperf\HttpServer\Annotation\Middleware;
use Hyperf\HttpServer\Annotation\PostMapping;
use Hyperf\Stringable\Str;

#[Controller(prefix: '/api/admin')]
#[Middleware(AdminMiddleware::class)]
class ApiController
{
    #[Inject]
    protected Upgrading $service;

    // sforum github api接口
    private string $api_releases = '/repos/zhuchunshu/SForum/releases';

    // sforum 通知接口
    private string $api_notice = 'https://www.runpod.cn/api/v1/SFService/sforum/admin/notice';

    // sforum 通知获取数量
    private int $api_notice_limit = 100;

    public function __construct()
    {
        $this->api_releases = get_options('github_api_url', 'https://api.github.com') . $this->api_releases;
    }

    #[PostMapping('getVersion')]
    public function getVersion()
    {
        // 获取最新版
        if (! cache()->has('admin.git.getVersion')) {
            $data = http()->get($this->api_releases);
            // 当前版本
            $current = null;
            foreach ($data as $item) {
                if (arr_has($item, 'tag_name') && $item['tag_name'] === build_info()->version) {
                    $current = $item;
                }
            }
            $data = $data[0];
            // 获取当前程序版本信息
            $build_info = (include BASE_PATH . '/build-info.php');
            $data = array_merge($data, $build_info);
            $version = $data['version'];
            $tag_name = $data['tag_name'];
            // 判断是否可升级
            $data['upgrade'] = false;
            if ($tag_name > $version && $data['prerelease'] === false) {
                $data['upgrade'] = true;
            }
            // 其他
            // 最新版本url
            $data['new_version_url'] = '/admin/Releases/' . $data['id'];
            $data['current_version_url'] = null;
            if ($current) {
                $data['current_version_url'] = '/admin/Releases/current/' . $current['id'];
            }
            // 返回数据
            cache()->set('admin.git.getVersion', $data, 600);
        }
        return cache()->get('admin.git.getVersion');
    }

    #[PostMapping('getRelease/{id}')]
    public function getRelease($id)
    {
        if (! cache()->has('admin.git.getRelease.' . $id)) {
            $data = http()->get($this->api_releases . '/' . $id);
            $r = http('raw')->get($data['html_url']);
            $body = $r->getBody();
            $body = Str::after($body, '<div data-pjax="true" data-test-selector="body-content" data-view-component="true" class="markdown-body my-3">');
            $body = Str::before($body, '</div>');
            $data['body'] = $body;
            cache()->set('admin.git.getRelease.' . $id, $data, 600);
        }
        return cache()->get('admin.git.getRelease.' . $id);
    }

    #[PostMapping('getUpdateLog')]
    public function getCommit()
    {
        return (new ContentParse())->parse('此功能已关闭');
    }

    #[PostMapping('clearCache')]
    public function clearCache()
    {
        cache()->delete('admin.git.getVersion');
        return Json_Api(200, true, ['msg' => '缓存清理成功']);
    }

    #[PostMapping('update')]
    public function update()
    {
        $url = match ((string) get_options('update_server', 2)) {
            '2' => '',
            '1' => 'https://gh-proxy.com/',
        };
        $data = http()->get($this->api_releases);
        $data = $data[0];
        // 获取当前程序版本信息
        $build_info = (include BASE_PATH . '/build-info.php');
        $data = array_merge($data, $build_info);
        $version = $data['version'];
        $tag_name = $data['tag_name'];
        // 判断是否不可升级
        if ($tag_name <= $version || $data['prerelease'] === true) {
            return Json_Api(403, false, ['msg' => '无需升级!']);
        }
        // 生成文件下载链接
        $url .= 'https://github.com/zhuchunshu/SForum/archive/' . $tag_name . '.zip';
        // 定义文件存放路径
        $file_path = BASE_PATH . '/runtime/update.zip';
        // 创建下载任务
        $this->service->handle($url, $file_path);
        return Json_Api(200, true, ['msg' => '升级任务已创建']);
    }

    // 同意免责声明
    // 同意免责声明

    #[GetMapping('agree.disclaimer')]
    public function agree_disclaimer()
    {
        cache()->set('admin.core.disclaimer', time());
        return Json_Api(200, true, ['msg' => 'success']);
    }

    // 获取sforum通知

    #[PostMapping('sforum.notice')]
    public function sforum_notice(): array
    {
        try {
            $data = http()->post($this->api_notice, [
                'limit' => $this->api_notice_limit,
            ]);
        } catch (\Throwable) {
            return Json_Api(500, false, ['msg' => '获取失败']);
        }
        return Json_Api(200, true, ['msg' => 'success', 'data' => $data]);
    }
}
