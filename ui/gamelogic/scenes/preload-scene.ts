import Phaser from 'phaser';
import { Images, Scenes, Sounds } from "../enums";

export class PreloadScene extends Phaser.Scene {
    //title: Phaser.GameObjects.Text;
    //hint: Phaser.GameObjects.Text;
    // music: Phaser.Sound.BaseSound;

    constructor() {
        super({
            key: Scenes.Preload
        });
    }

    preload() {
        let progressBar = this.add.graphics();
        let progressBox = this.add.graphics();
        progressBox.fillStyle(0x222222, 0.8);
        progressBox.fillRect(240, 270, 320, 50);

        let width = this.cameras.main.width;
        let height = this.cameras.main.height;
        let loadingText = this.make.text({
            x: width / 2,
            y: height / 2 - 50,
            text: 'Loading...',
            style: {
                font: '20px monospace',
                color: '#000000'
            }
        });
        loadingText.setOrigin(0.5, 0.5);

        var percentText = this.make.text({
            x: width / 2,
            y: height / 2 - 5,
            text: '0%',
            style: {
                font: '18px monospace',
                color: '#000000'
            }
        });
        percentText.setOrigin(0.5, 0.5);

        var assetText = this.make.text({
            x: width / 2,
            y: height / 2 + 50,
            text: '',
            style: {
                font: '18px monospace',
                color: '#000000'
            }
        });
        assetText.setOrigin(0.5, 0.5);

        this.load.on('progress', (value: any) => {
            console.log(value);
            progressBar.clear();
            progressBar.fillStyle(0xffffff, 1);
            progressBar.fillRect(250, 280, 300 * value, 30);
            percentText.setText(`${(value * 100).toFixed(2)} %`);
        });
                    
        this.load.on('fileprogress', (file: any) => {
            assetText.setText('Loading asset: ' + file.key);
        });
        
        this.load.on('complete', function () {
            progressBar.destroy();
            progressBox.destroy();
            loadingText.destroy();
            percentText.destroy();
            assetText.destroy();
        });

        this.load.image(Images.Tiles, 'assets/gamejam-brno-tiles.png');
        this.load.audio(Sounds.Theme, 'assets/soundtrack.mp3');
    }

    create() {
        this.scene.start(Scenes.Welcome);
    }
}