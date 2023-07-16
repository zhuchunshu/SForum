<?php

use Hyperf\Database\Schema\Schema;
use Hyperf\Database\Schema\Blueprint;
use Hyperf\Database\Migrations\Migration;

class UpdateTopicUpdatedTableAddUserAgentIp extends Migration
{
    /**
     * Run the migrations.
     */
    public function up(): void
    {
        Schema::table('topic_updated', function (Blueprint $table) {
			$table->string('user_agent')->nullable();
			$table->string('user_ip')->nullable();
        });
    }

    /**
     * Reverse the migrations.
     */
    public function down(): void
    {
        Schema::table('topic_updated', function (Blueprint $table) {
            //
        });
    }
}
