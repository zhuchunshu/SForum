@extends("App::app")

@section('title','主题已被删除')
@section('content')

    <div class="row row-cards justify-content-center">
        <div class="col-12">
            <div class="card">
                <div class="card-body">
                    <h3 class="card-title">主题已被删除</h3>
                    @php($quanxian=false)
                    @if(auth()->id()=== (int)$data->id && Authority()->check('topic_recover'))
                        @php($quanxian=true)
                    @elseif(Authority()->check('admin_topic_recover'))
                        @php($quanxian=true)
                    @elseif(\App\Plugins\Topic\src\Models\Moderator::query()->where('tag_id', $data->tag_id)->where('user_id',auth()->id())->exists())
                        @php($quanxian=true)
                    @endif
                    @if($quanxian===true)
                        {{--                        回收主题--}}
                        <form action="/topic/{{$data->id}}/topic.trashed.restore" method="post">
                            <x-csrf/>
                            <button type="submit" class="btn btn-primary">恢复主题</button>
                        </form>
                    @endif
                </div>
            </div>
        </div>
    </div>

@endsection