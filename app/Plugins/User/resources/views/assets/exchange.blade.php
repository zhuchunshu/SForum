<div class="row row-cards">
    <div class="col-md-12">
        <div class="card border-0">
            <div class="card-header">
                <h3 class="card-title">兑换</h3>
            </div>
            <div class="card-body" id="user-data-exchange">
                <form action="/" method="post" @@submit.prevent="submit">
                    <div class="mb-3">
                        <label for="" class="form-label">选择兑换类型</label>
                        <select class="form-select" v-model="selected">
                            @foreach(Itf()->get('user_exchange') as $key=>$data)
                                @if(!@$data['on'])
                                    <option value="{{$key}}">{{call_user_func($data['name'])}}</option>
                                @elseif(call_user_func($data['on']))
                                    <option value="{{$key}}">{{call_user_func($data['name'])}}</option>
                                @endif
                            @endforeach
                        </select>
                    </div>
                    @foreach(Itf()->get('user_exchange') as $key=>$data)
                        <div class="card card-body mb-3" v-if="selected==='{{$key}}'">
                            @if(!@$data['on'])
                                @include($data['view'])
                            @elseif(call_user_func($data['on']))
                                @include($data['view'])
                            @endif
                        </div>
                    @endforeach
                    <div class="mb-3">
                        <label for="" class="form-label">验证码</label>
                        <div class="input-group">
                            <input type="text" v-model="captcha" class="form-control" placeholder="captcha"
                                   autocomplete="off" required>
                            <span class="input-group-link">
                        <img class="captcha" src="{{captcha()->inline()}}" alt=""
                             onclick="this.src='/captcha?id='+Math.random()">
                    </span>
                        </div>
                    </div>

                    <button class="btn btn-primary">提交</button>
                </form>
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