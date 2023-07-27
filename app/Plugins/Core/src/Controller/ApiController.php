<?php

declare(strict_types=1);
/**
 * This file is part of zhuchunshu.
 * @link     https://github.com/zhuchunshu
 * @document https://github.com/zhuchunshu/SForum
 * @contact  laravel@88.com
 * @license  https://github.com/zhuchunshu/SForum/blob/master/LICENSE
 */
namespace App\Plugins\Core\src\Controller;

use App\Plugins\Comment\src\Model\TopicComment;
use App\Plugins\Core\src\Models\Report;
use App\Plugins\Core\src\Request\Report\CreateRequest;
use App\Plugins\Topic\src\Models\Topic;
use Hyperf\HttpServer\Annotation\Controller;
use Hyperf\HttpServer\Annotation\PostMapping;
use Hyperf\HttpServer\Annotation\RequestMapping;
use Hyperf\RateLimit\Annotation\RateLimit;

#[Controller(prefix: '/api/core')]
#[RateLimit(create: 1, capacity: 3)]
class ApiController
{
    // 创建举报

    #[PostMapping(path: 'report/create')]
    public function report_create(CreateRequest $request)
    {
        if (! auth()->check()) {
            return Json_Api(419, false, ['未登录']);
        }

        // 鉴权
        $quanxian = false;
        if (($request->input('type') === 'comment') && Authority()->check('report_comment')) {
            $quanxian = true;
        }
        if (($request->input('type') === 'topic') && Authority()->check('report_topic')) {
            $quanxian = true;
        }

        if ($quanxian === false) {
            return Json_Api(419, false, ['无权限']);
        }

        // 获取被举报内容的post_id;
        if (($request->input('type') === 'topic') && Topic::where('id', $request->input('type_id'))->exists()) {
            $post_id = Topic::where('id', $request->input('type_id'))->value('post_id');
        } elseif (($request->input('type') === 'comment') && TopicComment::where('id', $request->input('type_id'))->exists()) {
            $post_id = TopicComment::where('id', $request->input('type_id'))->value('post_id');
        } else {
            return Json_Api(419, false, ['无效的举报内容']);
        }

        if (Report::query()->where(['user_id' => auth()->id(), 'post_id' => $post_id])->exists()) {
            return Json_Api(403, false, ['你已举报此贴,无需重复举报']);
        }
        $content = '**违规页面地址:** ' . $request->input('url') . '
**举报原因:** ' . $request->input('report_reason') . "\n\n" . $request->input('content');
        $data = Report::query()->create([
            'post_id' => $post_id,
            'user_id' => auth()->id(),
            'title' => $request->input('title'),
            'content' => xss()->clean(markdown()->text($content)),
        ]);

        // 发送通知
        $users = [];
        foreach (Authority()->getUsers('admin_report') as $user) {
            $users[] = $user->id;
        }
        $mail_content = view('App::report.send_admin', ['data' => $data]);
        user_notice()->sends($users, '有用户举报了一条内容,需要你来审核', $mail_content, '/report/' . $data->id . '.html', true, 'system');
        return Json_Api(200, true, ['举报成功! 等待管理员审核']);
    }

    // 获取举报信息

    #[PostMapping(path: 'report/data')]
    public function report_data()
    {
        $report_id = request()->input('report_id');
        if (! $report_id) {
            return Json_Api(403, false, ['请求参数不足,缺少:report_id']);
        }
        if (! Report::query()->where('id', $report_id)->exists()) {
            return Json_Api(403, false, ['id为' . $report_id . '的举报内容不存在']);
        }
        $data = Report::query()->find($report_id, ['status']);
        return Json_Api(200, true, $data);
    }

    // 获取所有被举报并批准的的评论

    #[PostMapping(path: 'report/approve.comment')]
    public function report_approve_comment_list()
    {
//        $arr = [];
//        foreach (Report::query()->where(['status' => 'approve'])->get() as $value) {
//            $arr[] = $value->_id;
//        }
        return Json_Api(200, true, []);
    }

    #[PostMapping(path: 'report/update')]
    public function report_update()
    {
        if (! auth()->check() || ! Authority()->check('admin_report')) {
            return Json_Api(419, false, ['无权限']);
        }
        $report_id = request()->input('report_id');
        if (! $report_id) {
            return Json_Api(403, false, ['请求参数不足,缺少:report_id']);
        }
        if (! Report::query()->where('id', $report_id)->exists()) {
            return Json_Api(403, false, ['id为' . $report_id . '的举报内容不存在']);
        }
        $status = Report::query()->where('id', $report_id)->value('status');
        if ($status === 'pending') {
            $_status = 'approve';
            $_text = '批准';
        }
        if ($status === 'reject') {
            $_status = 'approve';
            $_text = '批准';
        }
        if ($status === 'approve') {
            $_status = 'reject';
            $_text = '驳回';
        }
        Report::query()->where('id', $report_id)->update([
            'status' => $_status,
        ]);

        if ($_status === 'approve') {
            $post_status = 'report';
        } else {
            $post_status = 'publish';
        }

        go(function () use ($report_id, $post_status) {
            $report = Report::find($report_id);
            $post = \App\Plugins\Core\src\Models\Post::find($report->post_id);
            if ($post->comment_id) {
                // 评论
                \Hyperf\DbConnection\Db::table('topic_comment')->where('id', $post->comment_id)->update([
                    'status' => $post_status,
                ]);
            } else {
                if ($post->topic_id) {
                    // 主题
                    \Hyperf\DbConnection\Db::table('topic')->where('id', $post->topic_id)->update([
                        'status' => $post_status,
                    ]);
                }
            }
        });
        return Json_Api(200, true, [$_text . '成功!']);
    }

    // 删除举报

    #[PostMapping(path: 'report/remove')]
    public function report_remove()
    {
        if (! auth()->check() || ! Authority()->check('admin_report')) {
            return Json_Api(419, false, ['无权限']);
        }
        $report_id = request()->input('report_id');
        if (! $report_id) {
            return Json_Api(403, false, ['请求参数不足,缺少:report_id']);
        }
        if (! Report::query()->where('id', $report_id)->exists()) {
            return Json_Api(403, false, ['id为' . $report_id . '的举报内容不存在']);
        }

        // 举报快照
        $data = Report::query()->where('id', $report_id)->first();

        Report::query()->where('id', $report_id)->delete();
        go(function () use ($report_id, ) {
            $report = Report::find($report_id);
            $post = \App\Plugins\Core\src\Models\Post::find($report->post_id);
            if ($post->comment_id) {
                // 评论
                \Hyperf\DbConnection\Db::table('topic_comment')->where('id', $post->comment_id)->update([
                    'status' => 'publish',
                ]);
            } else {
                if ($post->topic_id) {
                    // 主题
                    \Hyperf\DbConnection\Db::table('topic')->where('id', $post->topic_id)->update([
                        'status' => 'publish',
                    ]);
                }
            }
        });

        // 发送通知
        $users = [];
        foreach (Authority()->getUsers('admin_report') as $user) {
            $users[] = $user->id;
        }

        $user_data = auth()->data();
        $mail_content = view('App::report.remove_admin', ['data' => $data, 'user' => $user_data]);

        user_notice()->sends($users, '有管理员删除了一条举报,特此通知!', $mail_content, url('/'), true, 'system');
        return Json_Api(200, true, ['删除成功!']);
    }

    // 切换主题

    #[RateLimit(create: 1, capacity: 1)]
    #[PostMapping(path: 'toggle.theme')]
    public function theme_toggle()
    {
        if (! request()->input('theme')) {
            return Json_Api(403, false, ['msg' => '请求参数不足,缺少:theme']);
        }
        session()->set('theme', request()->input('theme'));
        session()->set('bs_theme', request()->input('bs_theme'));
        return Json_Api(200, true, ['msg' => '切换成功!']);
    }

    // 切换主题

    #[RateLimit(create: 1, capacity: 1)]
    #[PostMapping(path: 'toggle.auto.theme')]
    public function auto_theme_toggle()
    {
        if (! request()->input('theme')) {
            return Json_Api(403, false, ['msg' => '请求参数不足,缺少:theme']);
        }
        session()->set('auto_theme', request()->input('theme'));
        return Json_Api(200, true, ['msg' => '切换成功!']);
    }

    // 获取所有emoji

    #[RequestMapping(path: 'OwO.json')]
    public function emoji()
    {
        return (new \App\Plugins\Core\src\Lib\Emoji())->get();
    }

    #[RequestMapping(path: 'qr_code')]
    public function qr_code(): \Psr\Http\Message\ResponseInterface
    {
        $content = request()->input('content', url());
        $qr_code = qr_code()->format('svg')->generate($content);
        return response()->raw($qr_code)->withHeader('Content-Type', 'image/svg+xml');
    }

    #[RequestMapping(path: 'useragentinfo')]
    public function useragentinfo()
    {
        return [
            'useragent' => get_user_agent(),
            'ip' => get_client_ip(),
            'ip_info' => get_client_ip_data(get_client_ip()),
        ];
    }
}
