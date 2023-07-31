<?php

declare (strict_types=1);
/**
 * This file is part of zhuchunshu.
 * @link     https://github.com/zhuchunshu
 * @document https://github.com/zhuchunshu/SForum
 * @contact  laravel@88.com
 * @license  https://github.com/zhuchunshu/SForum/blob/master/LICENSE
 */
namespace App\Plugins\Core\src\Controller\Pay;

use App\Middleware\AdminMiddleware;
use App\Plugins\Core\src\Handler\FileUpload;
use App\Plugins\Core\src\Models\PayConfig;
use App\Plugins\Core\src\Models\PayOrder;
use Hyperf\HttpServer\Annotation\Controller;
use Hyperf\HttpServer\Annotation\GetMapping;
use Hyperf\HttpServer\Annotation\Middleware;
use Hyperf\HttpServer\Annotation\PostMapping;
#[Controller(prefix: '/admin/Pay')]
#[Middleware(AdminMiddleware::class)]
class PayAdminController
{
    /**
     * 订单.
     * @return \Psr\Http\Message\ResponseInterface
     */
    #[GetMapping('')]
    public function index()
    {
        $where_column = request()->input('where_column');
        $q = request()->input('trade_no');
        if (!$where_column) {
            $_orderBy = 'ASC';
            if ($q) {
                $page = PayOrder::query()->where('trade_no', 'like', '%' . $q . '%')->orWhere('id', 'like', '%' . $q . '%')->paginate(15);
            } else {
                $page = PayOrder::query()->paginate(15);
            }
            return view('App::Pay.admin.index', ['page' => $page, '_orderBy' => $_orderBy]);
        }
        $_orderBy = match (request()->input('_orderBy')) {
            'ASC' => 'DESC',
            'DESC' => 'ASC',
        };
        if ($q) {
            $page = PayOrder::query()->where('trade_no', 'like', '%' . $q . '%')->orWhere('id', 'like', '%' . $q . '%')->orderBy($where_column, $_orderBy)->paginate(15);
        } else {
            $page = PayOrder::query()->orderBy($where_column, $_orderBy)->paginate(15);
        }
        return view('App::Pay.admin.index', ['page' => $page, '_orderBy' => $_orderBy]);
    }
    /**
     * 配置.
     */
    #[GetMapping('config')]
    public function config()
    {
        return view('App::Pay.admin.config');
    }
    #[PostMapping('config')]
    public function config_save(FileUpload $fileUpload)
    {
        // 先处理上传的文件
        foreach (request()->getUploadedFiles() as $name => $file) {
            // 获取文件
            $file = request()->file($name);
            if ($file->isFile()) {
                // 保存文件
                $file = $fileUpload->save($file, 'admin', 'pay_');
                // 获取文件路径
                $file = $file['raw_path'];
                // 插入到数据库
                $this->config_save_insert($name, $file);
            }
        }
        // 后处理保存的字符串内容
        foreach (request()->all() as $key => $value) {
            $this->config_save_insert($key, $value);
        }
        pay()->clean_options();
        return redirect()->url('/admin/Pay/config')->with('success', '更新成功')->go();
    }
    /**
     * 支付设置.
     * @return \Psr\Http\Message\ResponseInterface
     */
    #[GetMapping('setting')]
    public function setting()
    {
        return view('App::Pay.admin.setting');
    }
    #[PostMapping('setting')]
    public function setting_submit()
    {
        $pay = request()->input('pay', []);
        $pay = json_encode($pay, JSON_UNESCAPED_UNICODE);
        $this->config_save_insert('enable', $pay);
        pay()->clean_options();
        return redirect()->url('/admin/Pay/setting')->with('success', '更新成功')->go();
        //return view('App::Pay.admin.setting');
    }
    #[GetMapping('{trade_no}/order')]
    public function order_show($trade_no)
    {
        if (!PayOrder::query()->where('trade_no', $trade_no)->exists()) {
            return redirect()->url('/admin/Pay')->with('danger', '订单不存在')->go();
        }
        return pay()->find($trade_no);
    }
    private function config_save_insert($name, $value) : bool
    {
        if (PayConfig::query()->where('name', $name)->exists()) {
            // 存在则更新
            PayConfig::query()->where('name', $name)->update(['value' => $value]);
            return true;
        }
        // 不存在则新建
        PayConfig::query()->create(['value' => $value, 'name' => $name]);
        return true;
    }
}