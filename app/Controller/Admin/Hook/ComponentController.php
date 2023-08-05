<?php

declare(strict_types=1);
/**
 * This file is part of zhuchunshu.
 * @link     https://github.com/zhuchunshu
 * @document https://github.com/zhuchunshu/SForum
 * @contact  laravel@88.com
 * @license  https://github.com/zhuchunshu/SForum/blob/master/LICENSE
 */

namespace App\Controller\Admin\Hook;

use App\Middleware\AdminMiddleware;
use Hyperf\HttpServer\Annotation\Controller;
use Hyperf\HttpServer\Annotation\GetMapping;
use Hyperf\HttpServer\Annotation\Middleware;
use Hyperf\HttpServer\Annotation\PostMapping;
use Hyperf\Paginator\LengthAwarePaginator;
use Hyperf\Utils\Collection;
use Hyperf\Stringable\Str;
use Hyperf\ViewEngine\Contract\FactoryInterface;
use Swoole\Coroutine\System;
use Symfony\Component\Finder\Finder;

#[Controller(prefix: '/admin/hook/components')]
#[Middleware(AdminMiddleware::class)]
class ComponentController
{
    /**
     * 部件列表.
     * @return \Psr\Http\Message\ResponseInterface
     */
    #[GetMapping('')]
    public function index()
    {
        if (!is_dir(BASE_PATH . '/resources/views/customize/component')) {
            System::exec('cd ' . BASE_PATH . '/resources/views' . '&& mkdir ' . 'customize && cd customize && mkdir component');
        }
        $page = $this->page();
        return view('admin.setting.hook.components.index', ['page' => $page]);
    }

    /**
     * 预览部件.
     * @return \Psr\Http\Message\ResponseInterface
     * @throws \Psr\Container\ContainerExceptionInterface
     * @throws \Psr\Container\NotFoundExceptionInterface
     */
    #[GetMapping('preview')]
    public function preview()
    {
        $component = request()->input('component');
        $component = 'customize.component.' . $component;
        $container = \Hyperf\Context\ApplicationContext::getContainer();
        $factory = $container->get(FactoryInterface::class);
        if (!$factory->exists($component)) {
            return admin_abort('小部件不存在', 403);
        }
        return view('admin.setting.hook.components.preview', ['view' => $component]);
    }

    /**
     * 修改小部件代码
     * @return \Psr\Http\Message\ResponseInterface
     */
    #[GetMapping('edit')]
    public function edit()
    {
        $path = request()->input('path');
        $path = BASE_PATH . '/resources/views/customize/component/' . $path;
        if (!file_exists($path)) {
            return admin_abort('文件不存在');
        }
        return view('admin.setting.hook.components.edit', ['view' => $path, 'path' => request()->input('path')]);
    }

    /**
     * 获取小部件文件内容.
     * @return array|\Psr\Http\Message\ResponseInterface
     */
    #[GetMapping('get_file_content')]
    public function get_file_content()
    {
        $path = request()->input('path');
        $path = BASE_PATH . '/resources/views/customize/component/' . $path;
        if (!file_exists($path)) {
            return admin_abort('文件不存在');
        }
        return Json_Api(200, true, ['msg' => '获取成功!', 'content' => file_get_contents($path,)]);
    }

    /**
     * 提交修改部件代码
     * @return array|\Psr\Http\Message\ResponseInterface
     */
    #[PostMapping('put_file_content')]
    public function put_file_content()
    {
        $path = request()->input('path');
        $path = BASE_PATH . '/resources/views/customize/component/' . $path;
        if (!file_exists($path)) {
            return admin_abort('文件不存在');
        }
        file_put_contents($path, request()->input('content'));
        return Json_Api(200, true, ['msg' => '修改成功!']);
    }

    /**
     * 创建小部件.
     */
    #[GetMapping('create')]
    public function create(): \Psr\Http\Message\ResponseInterface
    {
        return view('admin.setting.hook.components.create');
    }

    #[PostMapping('create')]
    public function store()
    {
        $content = request()->input('content');
        $name = request()->input('name');
        if (!$content || !$name) {
            return admin_abort('请求参数不完整');
        }
        if (!preg_match('/[a-zA-Z0-9.]/i', $name)) {
            return admin_abort('小部件名称格式有误,只支持字母、数字和符号点.');
        }
        if (in_array($name, $this->get_all_components_filename())) {
            return admin_abort('此部件名称已存在，换一个试试吧');
        }
        $_file = explode('.', $name);
        $file_name = array_pop($_file);
        $root_dir = BASE_PATH . '/resources/views/customize/component';
        if (count($_file) >= 1) {
            unset($_file[$file_name]);
            foreach ($_file as $dirName) {
                if (!is_dir(!$root_dir . '/' . $dirName)) {
                    System::exec('cd ' . $root_dir . ' && mkdir ' . $dirName);
                }
                $root_dir = $root_dir . "/" . $dirName;
            }
        }
        $file_path = $root_dir . "/" . $file_name . '.blade.php';
        $_file_path = Str::after($file_path, BASE_PATH . '/resources/views/customize/component');
        file_put_contents($file_path, $content);
        return Json_Api(200, true, ['msg' => '创建成功!', 'redirect' => '/admin/hook/components/edit?path=' . $_file_path]);
    }

    #[PostMapping("delete")]
    public function delete(){
        $path = request()->input('path');
        $path = BASE_PATH . '/resources/views/customize/component/' . $path;
        if (!file_exists($path)) {
            return admin_abort('文件不存在');
        }
        System::exec('rm -rf '.$path);
        return Json_Api(200,true,['msg' => '删除成功!']);
    }

    /**
     * 获取所有小部件的名称
     * @return array
     */
    public function get_all_components_filename()
    {
        $result = [];
        foreach ($this->getAllComponents() as $data) {
            $result[] = $data['import'];
        }
        return $result;
    }

    /**
     * 获取所有小部件.
     */
    private function getAllComponents(): array
    {
        $path = BASE_PATH . '/resources/views/customize/component/';
        $files = [];
        $result = Finder::create()->in($path)->files()->name('*.blade.php');
        $id = 1;
        foreach ($result as $item) {
            if($item->getRelativePath()){
                $Filename = $item->getRelativePath() . '/' . $item->getFilename();
            }else{
                $Filename = $item->getFilename();
            }
            $viewName = Str::before($item->getFilename(), '.');
            $RelativePath = $item->getRelativePath();
            if ($RelativePath) {
                $RelativePath = explode('/', $RelativePath);
                $viewName = implode('.', $RelativePath) . '.' . $viewName;
            }
            $files[] = [
                'id' => $id++,
                'file_name' => $Filename,
                'view' => 'customize.component.' . $viewName,
                'import' => $viewName,
                'path' => $item->getRealPath(),
                'remark' => $this->get_remark($item->getRealPath()) ?: '暂无',
            ];
        }
        return array_values($files);
    }

    /**
     * 分页.
     */
    private function page(): LengthAwarePaginator
    {
        $currentPage = (int)request()->input('page', 1);
        $perPage = (int)request()->input('per_page', 15);

        // 这里根据 $currentPage 和 $perPage 进行数据查询，以下使用 Collection 代替
        $collection = new Collection($this->getAllComponents());

        $data = array_values($collection->forPage($currentPage, $perPage)->toArray());
        return new LengthAwarePaginator($data, count($this->getAllComponents()), $perPage, $currentPage);
    }

    /**
     * get remark.
     * @param $file
     * @return null|string
     */
    private function get_remark($file): string|null
    {
        $content = fopen($file, 'rb');
        $content = fgets($content);
        $result = [];
        preg_match_all('/(?:\\{{--)(.*)(?:\\--}})/i', $content, $result);
        return @$result[1][0] ?: null;
    }
}
