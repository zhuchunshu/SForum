<?php

namespace App\Plugins\Topic\src\Handler\Topic;

use App\Plugins\Comment\src\Model\TopicComment;
use App\Plugins\Core\src\Models\Report;
use App\Plugins\Topic\src\Models\Topic;
use Hyperf\Utils\Str;

class ShowTopic
{
    public function handle($id,$comment_page)
    {
        if(Report::query()->where(['type' => 'topic','_id' => $id,'status' => 'approve'])->exists()){
            return admin_abort('此帖子已被举报并批准,无法查看',403);
        }
        // 自增浏览量
        $updated_at = Topic::query()->where('id', $id)->first()->updated_at;
        Topic::query()->where('id', $id)->increment('view',1,['updated_at' => $updated_at]);

        // 缓存
        $data = Topic::query(true)
            ->where('id', $id)
            ->with("tag","user","topic_updated","update_user")
            ->first();
        // 创建数据
        $shang = Topic::query()->where([['id','<',$id],['status','publish']])->select('title','id')->orderBy('id','desc')->first();
        $xia = Topic::query()->where([['id','>',$id],['status','publish']])->select('title','id')->orderBy('id','asc')->first();
        $sx = ['shang' => $shang,'xia' => $xia];
        $comment_count = TopicComment::query()->where(['status' => 'publish','topic_id'=>$id])->count();
        $this->session($data);
        $comment = null;
        // 评论分页数据
        if (get_options("comment_topic_show_type","default")==="default"){
            $comment = TopicComment::query()
                ->where(['status' => 'publish','topic_id'=>$id])
                ->with("topic","user","parent")
                ->orderBy("optimal","desc")
                ->orderBy("likes","desc")
                ->paginate(get_options("comment_page_count",15));
        }
        return view('Core::topic.show.show',['data' => $data,'get_topic' => $sx,'comment_count'=>$comment_count,'comment' => $comment,'comment_page' => $comment_page]);
    }

    public function session($data): void
    {
        if(!session()->has("view_topic_data")){
            session()->set("view_topic_data","view.topic.".Str::random());
        }
        cache()->set(session()->get("view_topic_data"),$data,600);
    }
}