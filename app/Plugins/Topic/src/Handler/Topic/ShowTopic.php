<?php

namespace App\Plugins\Topic\src\Handler\Topic;

use App\Plugins\Comment\src\Model\TopicComment;
use App\Plugins\Topic\src\Models\Topic;
use Hyperf\Utils\Str;

class ShowTopic
{
    public function handle($id)
    {
        // 自增浏览量
        $updated_at = Topic::query()->where('id', $id)->first()->updated_at;
        Topic::query()->where('id', $id)->increment('view',1,['updated_at' => $updated_at]);

        if(!cache()->has("topic.data.".$id)){
            $data = Topic::query()
                ->where('id', $id)
                ->with("tag","user","topic_updated","update_user")
                ->first();
            cache()->set("topic.data.".$id, $data);
        }else{
            $data = cache()->get("topic.data.".$id);
        }
        $shang = Topic::query()->where([['id','<',$id],['status','publish']])->select('title','id')->orderBy('id','desc')->first();
        $xia = Topic::query()->where([['id','>',$id],['status','publish']])->select('title','id')->orderBy('id','asc')->first();
        $sx = ['shang' => $shang,'xia' => $xia];
        $comment_count = TopicComment::query()->where(['status' => 'publish','topic_id'=>$id])->count();
        $this->session($data);
        $comment = null;
        if (get_options("comment_topic_show_type","default")==="default"){
            $comment = TopicComment::query()->where(['status' => 'publish','topic_id'=>$id])->paginate(get_options("comment_page_count",15));
        }
        return view('plugins.Core.topic.show.show',['data' => $data,'get_topic' => $sx,'comment_count'=>$comment_count,'comment' => $comment]);
    }

    public function session($data){
        if(!session()->has("view_topic_data")){
            session()->set("view_topic_data","view.topic.".Str::random());
        }
        if(!cache()->has(session()->get("view_topic_data"))){
            cache()->set(session()->get("view_topic_data"),$data,600);
        }
    }
}