<?php

declare(strict_types=1);
/**
 * This file is part of zhuchunshu.
 * @link     https://github.com/zhuchunshu
 * @document https://github.com/zhuchunshu/super-forum
 * @contact  laravel@88.com
 * @license  https://github.com/zhuchunshu/super-forum/blob/master/LICENSE
 */

namespace App\Controller\Admin;

use App\Controller\AbstractController;
use App\Middleware\AdminMiddleware;
use Hyperf\HttpServer\Annotation\Controller;
use Hyperf\HttpServer\Annotation\GetMapping;
use Hyperf\HttpServer\Annotation\Middleware;
use Hyperf\HttpServer\Annotation\PostMapping;
use Hyperf\Paginator\LengthAwarePaginator;
use Hyperf\Utils\Collection;

#[Controller(prefix: '/admin/setting/menu')]
#[Middleware(AdminMiddleware::class)]
class MenuController extends AbstractController
{
    #[GetMapping(path: '')]
    public function index()
    {
        return view('admin.setting.menu.index', ['page' => $this->page()]);
    }

    #[GetMapping(path: 'create')]
    public function create()
    {
        return view('admin.setting.menu.create');
    }

    #[GetMapping(path: '{id}/edit')]
    public function edit($id)
    {
        if (!in_array($id, _menu_keys())) {
            return admin_abort('菜单不存在', 403);
        }
        $data = _menu_get_data($id);
        if (arr_has($data, 'quanxian')) {
            unset($data['quanxian']);
        }
        return view('admin.setting.menu.edit', ['data' => $data]);
    }

    #[PostMapping(path: 'create')]
    public function store()
    {
        $data = $this->request->input('data');
        $data = json_decode($data, true);
        if (arr_has($data, 'parent_id') && data_get($data, 'parent_id') && !in_array($data['parent_id'], _menu_keys())) {
            return Json_Api(403, false, ['msg' => '上级id不存在']);
        }
        $data['sort'] = (int)$data['sort'];
        if (arr_has($data, 'parent_id') && data_get($data, 'parent_id')) {
            $data['parent_id'] = (int)$data['parent_id'];
        } else {
            unset($data['parent_id']);
        }
        if (arr_has($data, 'parent_id') && data_get($data, 'parent_id') && arr_has(_menu_get_data($data['parent_id']), 'parent_id')) {
            return Json_Api(403, false, ['msg' => '子菜单不能作为上级菜单使用']);
        }
        $prefix_name = config('cache.default.prefix') . 'menu';
        redis()->hSetNx($prefix_name, (string)((int)max(_menu_keys()) + 1), _menu_instance()->serialize($data));
        return Json_Api(200, true, ['msg' => '创建成功!']);
    }

    #[PostMapping(path: 'update')]
    public function update()
    {
        $data = $this->request->input('data');
        $data = json_decode($data, true);
        if (!in_array($this->request->input('id'), _menu_keys())) {
            return Json_Api(403, false, ['msg' => '被修改的菜单id不存在']);
        }
        if (arr_has($data, 'parent_id') && data_get($data, 'parent_id') && !in_array($data['parent_id'], _menu_keys())) {
            return Json_Api(403, false, ['msg' => '上级id不存在']);
        }
        $data['sort'] = (int)$data['sort'];
        if (arr_has($data, 'parent_id') && data_get($data, 'parent_id')) {
            $data['parent_id'] = (int)$data['parent_id'];
        } else {
            unset($data['parent_id']);
        }
        if (arr_has($data, 'parent_id') && data_get($data, 'parent_id') && arr_has(_menu_get_data($data['parent_id']), 'parent_id')) {
            return Json_Api(403, false, ['msg' => '子菜单不能作为上级菜单使用']);
        }
        $_data = _menu_get_data($this->request->input('id'));
        $data = array_merge($_data, $data);
        if (arr_has($data, 'quanxian')) {
            $data['quanxian'] = _menu_instance()->serialize($data['quanxian']);
        }
        $prefix_name = config('cache.default.prefix') . 'menu';
        redis()->hDel($prefix_name, (string)$this->request->input('id'));
        redis()->hSetNx($prefix_name, (string)$this->request->input('id'), _menu_instance()->serialize($data));
        return Json_Api(200, true, ['msg' => '修改成功!']);
    }

    #[PostMapping(path: "delete")]
    public function delete()
    {
        $id = $this->request->input('id');
        if (!in_array($this->request->input('id'), _menu_keys())) {
            return Json_Api(403, false, ['msg' => '被删除的菜单id不存在']);
        }
        if (arr_has(Itf()->get('menu'), $id)) {
            return Json_Api(403, false, ['msg' => '这是不可删除的菜单']);
        }
        $prefix_name = config('cache.default.prefix') . 'menu';
        redis()->hDel($prefix_name, (string)$this->request->input('id'));
        return Json_Api(200, true, ['msg' => '删除成功!']);
    }

    private function page()
    {
        //_menu()
        $currentPage = (int)request()->input('page', 1);
        $perPage = (int)request()->input('per_page', 15);

        // 这里根据 $currentPage 和 $perPage 进行数据查询，以下使用 Collection 代替
        $collection = new Collection(_menu());

        $data = array_values($collection->forPage($currentPage, $perPage)->toArray());
        return new LengthAwarePaginator($data, count(_menu()), $perPage, $currentPage);
    }
}
