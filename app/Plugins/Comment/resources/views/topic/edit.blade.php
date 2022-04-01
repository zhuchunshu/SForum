@extends("Core::app")

@section('title',"修改id为:".$data->id."的评论")


@section('content')

   <div class="row justify-content-center">
       <div class="col-md-12">
           <div class="border-0 card">
               <div class="card-header">
                   <h3 class="card-title"><a class="text-red" href="/{{$data->topic->id}}.html">{{$data->topic->title}}</a> 下id为{{$data->id}}的评论</h3>
               </div>
               <div class="card-body" id="vue-comment-topic-edit-form">
                   <form action="" method="post" @@submit.prevent="submit">
                       <div id="vditor"></div>
                       <button class="btn btn-primary" style="margin-top: 5px">提交</button>
                   </form>
               </div>
           </div>
       </div>
   </div>

@endsection

@section('headers')
    <link rel="stylesheet" href="{{ mix('plugins/Topic/css/app.css') }}">
@endsection
@section('scripts')
    <script>var comment_id = {{$data->id}}</script>
    <script>var topic_id = {{$data->topic_id}}</script>
    <script src="{{mix("plugins/Comment/js/edit.js")}}"></script>
@endsection