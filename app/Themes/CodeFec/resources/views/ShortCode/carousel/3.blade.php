<div id="carousel-{{$id}}-thumb" class="carousel slide carousel-fade" data-bs-ride="carousel">
    <div class="carousel-indicators carousel-indicators-thumb">

        @foreach($images as $key=>$item)
            @if($loop->first)
                <button type="button" data-bs-target="#carousel-{{$id}}-thumb" data-bs-slide-to="{{$key}}"
                        class="ratio ratio-4x3 active" style="background-image: url({{$item->image}})"></button>
            @else
                <button type="button" data-bs-target="#carousel-{{$id}}-thumb" data-bs-slide-to="{{$key}}"
                        class="ratio ratio-4x3" style="background-image: url({{$item->image}})"></button>
            @endif
        @endforeach

    </div>
    <div class="carousel-inner {{$class}}" style="{{$style}}">
        @foreach($images as $key => $item)
            @if($loop->first)
                @if(@$item->url)
                    <a href="{{$item->url}}"> @endif
                        <div class="carousel-item active">
                            <img class="d-block w-100" alt=""
                                 src="{{$item->image}}"/>
                            @if(@$item->title || @$item->description)
                                <div class="carousel-caption-background d-none d-md-block"></div>
                                <div class="carousel-caption d-none d-md-block">
                                    @if(@$item->title)
                                        <h3>{{$item->title}}</h3>
                                    @endif
                                    @if(@$item->description)
                                        <p>{{$item->description}}</p>
                                    @endif
                                </div>
                            @endif
                        </div>
                        @if(@$item->url) </a>
                @endif

            @else
                @if(@$item->url)
                    <a href="{{$item->url}}"> @endif
                        <div class="carousel-item">
                            <img class="d-block w-100" alt="" src="{{$item->image}}"/>
                            @if(@$item->title || @$item->description)
                                <div class="carousel-caption-background d-none d-md-block"></div>
                                <div class="carousel-caption d-none d-md-block">
                                    @if(@$item->title)
                                        <h3>{{$item->title}}</h3>
                                    @endif
                                    @if(@$item->description)
                                        <p>{{$item->description}}</p>
                                    @endif
                                </div>
                            @endif
                        </div>
                        @if(@$item->url) </a>
                @endif
            @endif
        @endforeach
    </div>
</div>
