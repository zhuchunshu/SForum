<?php

namespace App\Plugins\Core\src\Controller\InvitationCode;

use App\Middleware\AdminMiddleware;
use App\Plugins\Core\src\Jobs\CreateInvitationCodeJob;
use App\Plugins\Core\src\Models\InvitationCode;
use Hyperf\Di\Annotation\Inject;
use Hyperf\HttpServer\Annotation\Controller;
use Hyperf\HttpServer\Annotation\GetMapping;
use Hyperf\HttpServer\Annotation\Middleware;
use Hyperf\HttpServer\Annotation\PostMapping;
#[Middleware(AdminMiddleware::class)]
#[Controller(prefix: "/admin/Invitation-code")]
class AdminController
{
    #[GetMapping("")]
    public function index()
    {
        $where = request()->input('where', 2);
        $page = match ((int) $where) {
            1 => InvitationCode::query()->orderByDesc('id')->paginate(30),
            2 => InvitationCode::query()->paginate(30),
            3 => InvitationCode::query()->where('status', true)->paginate(30),
        };
        return view("App::admin.InvitationCode.index", ['page' => $page]);
    }
    #[GetMapping("export")]
    public function export()
    {
        return view("App::admin.InvitationCode.export");
    }
    #[PostMapping("export")]
    public function export_submit()
    {
        $keywords = request()->input('keywords');
        $count = request()->input('count', 0);
        if ($count <= 0) {
            $data = InvitationCode::query()->where('code', 'like', '%' . $keywords . '%')->get();
        } else {
            $data = InvitationCode::query()->where('code', 'like', '%' . $keywords . '%')->limit($count)->get();
        }
        $content = null;
        foreach ($data as $value) {
            $content .= $value->code . "\n";
        }
        $path = BASE_PATH . "/app/CodeFec/storage/邀请码.txt";
        file_put_contents($path, $content);
        return response()->download($path);
    }
    #[PostMapping("")]
    public function index_post()
    {
        $where = request()->input('where', 2);
        $page = match ((int) $where) {
            1 => InvitationCode::query()->orderByDesc('id')->paginate(30),
            2 => InvitationCode::query()->paginate(30),
            3 => InvitationCode::query()->where('status', true)->paginate(30),
        };
        $data = [];
        foreach ($page->items() as $value) {
            $data[] = $value->id;
        }
        return $data;
    }
    #[GetMapping("create")]
    public function create()
    {
        return view("App::admin.InvitationCode.create");
    }
    /**
     * @var CreateInvitationCodeJob
     */
    #[Inject]
    protected CreateInvitationCodeJob $service;
    #[PostMapping("create")]
    public function create_submit()
    {
        $count = request()->input('count');
        $after = request()->input('after');
        $before = request()->input('before');
        if (!$count || !is_numeric($count)) {
            return redirect()->back()->with('danger', '生成数量有误')->go();
        }
        if ($count > 100000) {
            return redirect()->back()->with('danger', '一次最多生成10万个')->go();
        }
        $this->service->handle($count, $after, $before);
        return redirect()->back()->with('success', '任务已创建')->go();
    }
    #[PostMapping("remove")]
    public function remove()
    {
        $data = request()->input('data');
        if (!$data) {
            return Json_Api(403, false, ['msg' => '请求参数不足']);
        }
        if (!is_array($data)) {
            return Json_Api(403, false, ['msg' => '请求参数不足']);
        }
        foreach ($data as $id) {
            InvitationCode::query()->where('id', $id)->delete();
        }
        return Json_Api(200, true, ['msg' => '删除成功!']);
    }
}