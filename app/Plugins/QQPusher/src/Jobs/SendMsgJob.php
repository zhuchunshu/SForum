<?php

namespace App\Plugins\QQPusher\src\Jobs;

use App\Model\AdminOption;
use App\Plugins\QQPusher\src\GoCqhttp;
use App\Plugins\Topic\src\Models\Topic;
use Hyperf\AsyncQueue\Annotation\AsyncQueueMessage;
use Hyperf\AsyncQueue\Job;
use Noodlehaus\Config;

class SendMsgJob extends Job
{
    public string|int $topic_id;

    /**
     * 任务执行失败后的重试次数，即最大执行次数为 $maxAttempts+1 次
     *
     * @var int
     */
    protected $maxAttempts = 2;

    public function __construct($topic_id)
    {
        // 这里最好是普通数据，不要使用携带 IO 的对象，比如 PDO 对象
        $this->topic_id = $topic_id;
    }

    public function handle()
    {
        // 需要异步执行的代码逻辑
        // 这里的逻辑会在 ConsumerProcess 进程中执行
        $topic = Topic::query()->find($this->topic_id);
        $groups = Config::load(plugin_path('QQPusher/groups.json'))->all();
        foreach ($groups as $group_id) {

            GoCqhttp::post('send_group_msg', [
                'message' => '--- 论坛新帖通知 --- 
主题： ' . $topic->title . "\n链接：" . $this->url('/' . $topic->id . '.html') . "\n点击上方链接参与探讨。",
                'group_id' => $group_id,
            ]);
        }
    }

    private function url($path = null)
    {
        $url = $this->get_options("APP_URL", "http://127.0.0.1");
        if (!$path) {
            return $url;
        }
        return $url . $path;
    }

    private function get_options($name, $default = "")
    {
        if (!cache()->has('admin.options.' . $name)) {
            cache()->set("admin.options." . $name, @AdminOption::query()->where("name", $name)->first()->value);
        }
        return $this->core_default(cache()->get("admin.options." . $name), $default);
    }

    private function core_default($string = null, $default = null)
    {
        if ($string) {
            return $string;
        }
        return $default;
    }
}