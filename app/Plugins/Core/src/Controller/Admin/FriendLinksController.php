<?php

declare(strict_types=1);
/**
 * This file is part of zhuchunshu.
 * @link     https://github.com/zhuchunshu
 * @document https://github.com/zhuchunshu/SForum
 * @contact  laravel@88.com
 * @license  https://github.com/zhuchunshu/SForum/blob/master/LICENSE
 */
namespace App\Plugins\Core\src\Controller\Admin;

use App\Middleware\AdminMiddleware;
use App\Plugins\Core\src\Models\FriendLink;
use Hyperf\Di\Annotation\Inject;
use Hyperf\HttpServer\Annotation\Controller;
use Hyperf\HttpServer\Annotation\GetMapping;
use Hyperf\HttpServer\Annotation\Middleware;
use Hyperf\HttpServer\Annotation\PostMapping;
use Hyperf\Validation\Contract\ValidatorFactoryInterface;

#[Middleware(AdminMiddleware::class)]
#[Controller(prefix: '/admin/setting/friend_links')]
class FriendLinksController
{
    /**
     * @var ValidatorFactoryInterface
     */
    #[Inject]
    protected ValidatorFactoryInterface $validationFactory;

    #[GetMapping(path: '')]
    public function index()
    {
        $page = FriendLink::query()->orderByDesc('id')->paginate(12);
        return view('App::admin.FriendLinks.index', ['page' => $page]);
    }

    #[GetMapping(path: 'create')]
    public function create()
    {
        return view('App::admin.FriendLinks.create');
    }

    #[PostMapping(path: 'create')]
    public function store()
    {
        $validator = $this->validationFactory->make(
            request()->all(),
            [
                'name' => 'required',
                'link' => 'required',
                'icon' => 'nullable',
                'to_sort' => 'required|integer',
                '_blank' => 'required',
                'hidden' => 'required',
                'description' => 'nullable|string',
            ],
            [],
            [
                'name' => '友链名称',
                'link' => '链接',
                'icon' => '图标',
                'to_sort' => '排序',
                '_blank' => '是否新标签打开',
                'hidden' => '是否在首页隐藏',
                'description' => '网站描述',
            ]
        );

        if ($validator->fails()) {
            // Handle exception
            $errorMessage = $validator->errors()->first();
            return redirect()->back()->with('danger', $errorMessage)->go();
        }
        $validated = $validator->validated();
        if ($validated['_blank'] === '开启') {
            $validated['_blank'] = true;
        }
        if ($validated['hidden'] === '开启') {
            $validated['hidden'] = true;
        }else{
            $validated['hidden'] = false;
        }

        FriendLink::query()->create($validated);
        return redirect()->url('/admin/setting/friend_links')->with('success', '添加成功!')->go();
    }

    #[GetMapping('{id}/edit')]
    public function edit($id)
    {
        if (! FriendLink::query()->where('id', $id)->exists()) {
            return redirect()->back()->with('danger', '友链不存在')->go();
        }
        $data = FriendLink::query()->find($id);
        return view('App::admin.FriendLinks.edit', ['data' => $data]);
    }

    #[PostMapping('')]
    public function update()
    {
        $id = request()->input('id');
        if (! FriendLink::query()->where('id', $id)->exists()) {
            return redirect()->back()->with('danger', '友链不存在')->go();
        }
        if (request()->input('action') === 'delete') {
            FriendLink::query()->where('id', $id)->delete();
            return redirect()->with('success', '删除成功!')->url('/admin/setting/friend_links')->go();
        }
        $validator = $this->validationFactory->make(
            request()->all(),
            [
                'name' => 'required',
                'link' => 'required',
                'icon' => 'nullable',
                'to_sort' => 'required|integer',
                '_blank' => 'required',
                'hidden' => 'required',
                'description' => 'nullable|string',
            ],
            [],
            [
                'name' => '友链名称',
                'link' => '链接',
                'icon' => '图标',
                'to_sort' => '排序',
                '_blank' => '是否新标签打开',
                'hidden' => '是否在首页隐藏',
                'description' => '网站描述',
            ]
        );

        if ($validator->fails()) {
            // Handle exception
            $errorMessage = $validator->errors()->first();
            return redirect()->back()->with('danger', $errorMessage)->go();
        }
        $validated = $validator->validated();
        if ($validated['_blank'] === '开启') {
            $validated['_blank'] = true;
        }
        if ($validated['hidden'] === '开启') {
            $validated['hidden'] = true;
        }

        FriendLink::query()->where('id', $id)->update($validated);
        return redirect()->with('success', '更新成功!')->url('/admin/setting/friend_links')->go();
    }
}
