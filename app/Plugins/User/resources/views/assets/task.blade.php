<div class="row row-cards">
    <div class="col-md-12">
        <div class="card border-0">
            <div class="card-header">
                <h3 class="card-title">任务</h3>
                <div class="card-actions">
                    <a href="/user/asset/record">资产变更记录</a>
                </div>
            </div>
            <div class="list-group list-group-flush overflow-auto" style="max-height: 35rem">
                <div class="list-group-header sticky-top">系统任务</div>

                @foreach(Itf()->get('user_task_system') as $system)
                    @if(call_user_func($system['show'])===true)
                        <div class="list-group-item">
                            @include($system['view'])
                        </div>
                    @endif
                @endforeach

                <div class="list-group-header sticky-top">每日任务</div>
                @foreach(Itf()->get('user_task_daily') as $daily)
                    @if(call_user_func($daily['show'])===true)
                        <div class="list-group-item">
                            @include($daily['view'])
                        </div>
                    @endif
                @endforeach
            </div>
        </div>
    </div>
</div>

@section('scripts')
    @php($submit_url = [])
    @foreach(Itf()->get('user_exchange') as $key=>$data)
        @if(!@$data['on'])
            @php($submit_url[$key]=$data['url'])
        @elseif(call_user_func($data['on']))
            @php($submit_url[$key]=$data['url'])
        @endif
    @endforeach
    @foreach(Itf()->get('user_exchange') as $key=>$data)
        @if(!@$data['on'])
            <script>
                var user_exchange_selected = "{{$key}}";
            </script>
            @php($submit_url[$key]=$data['url'])
            @break
        @elseif(call_user_func($data['on']))
            <script>
                var user_exchange_selected = "{{$key}}";
            </script>
            @php($submit_url[$key]=$data['url'])
            @break
        @endif
    @endforeach
    <script>
        var submit_urls = @json($submit_url,128,256)
    </script>
    <script>
        var user_id = {{$user->id}}
    </script>
    <script src="{{mix("plugins/Core/js/user.js")}}"></script>
    <script src="{{mix('plugins/User/js/order.js')}}"></script>
@endsection